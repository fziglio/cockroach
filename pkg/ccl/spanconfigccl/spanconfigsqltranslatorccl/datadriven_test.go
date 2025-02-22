// Copyright 2021 The Cockroach Authors.
//
// Licensed as a CockroachDB Enterprise file under the Cockroach Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/cockroachdb/cockroach/blob/master/licenses/CCL.txt

package spanconfigsqltranslatorccl

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/base"
	_ "github.com/cockroachdb/cockroach/pkg/ccl/kvccl/kvtenantccl"
	_ "github.com/cockroachdb/cockroach/pkg/ccl/partitionccl"
	"github.com/cockroachdb/cockroach/pkg/config/zonepb"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/spanconfig"
	"github.com/cockroachdb/cockroach/pkg/spanconfig/spanconfigtestutils"
	"github.com/cockroachdb/cockroach/pkg/spanconfig/spanconfigtestutils/spanconfigtestcluster"
	"github.com/cockroachdb/cockroach/pkg/sql/catalog/descpb"
	"github.com/cockroachdb/cockroach/pkg/sql/catalog/tabledesc"
	"github.com/cockroachdb/cockroach/pkg/testutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/sqlutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/testcluster"
	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/datadriven"
	"github.com/stretchr/testify/require"
)

// TestDataDriven is a data-driven test for the spanconfig.SQLTranslator. It
// allows users to set up zone config hierarchies and validate their translation
// to SpanConfigs is as expected. Only fields that are different from the
// (default) RANGE DEFAULT are printed in the test output for readability. It
// offers the following commands:
//
// - "exec-sql"
//   Executes the input SQL query.
//
// - "query-sql"
//   Executes the input SQL query and prints the results.
//
// - "translate" [database=<str>] [table=<str>] [named-zone=<str>] [id=<int>]
//   Translates the SQL zone config state to the span config state starting
//   from the referenced object (named zone, database, database + table, or
//   descriptor id) as the root.
//
// - "full-translate"
//   Performs a full translation of the SQL zone config state to the implied
//   span config state.
//
// - "mark-table-offline" [database=<str>] [table=<str>]
//   Marks the given table as offline for testing purposes.
//
// - "mark-table-public" [database=<str>] [table=<str>]
//   Marks the given table as public.
//
func TestDataDriven(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	ctx := context.Background()

	scKnobs := &spanconfig.TestingKnobs{
		// Instead of relying on the GC job to wait out TTLs and clear out
		// descriptors, let's simply exclude dropped tables to simulate
		// descriptors no longer existing. See comment on
		// ExcludeDroppedDescriptorsFromLookup for more details.
		ExcludeDroppedDescriptorsFromLookup: true,
		// We run the reconciler manually in this test (through the span config
		// test cluster).
		ManagerDisableJobCreation: true,
	}
	datadriven.Walk(t, testutils.TestDataPath(t), func(t *testing.T, path string) {
		tc := testcluster.StartTestCluster(t, 1, base.TestClusterArgs{
			ServerArgs: base.TestServerArgs{
				Knobs: base.TestingKnobs{
					SpanConfig: scKnobs,
				},
			},
		})
		defer tc.Stopper().Stop(ctx)

		spanConfigTestCluster := spanconfigtestcluster.NewHandle(t, tc, scKnobs)
		defer spanConfigTestCluster.Cleanup()

		var tenant *spanconfigtestcluster.Tenant
		if strings.Contains(path, "tenant") {
			tenant = spanConfigTestCluster.InitializeTenant(ctx, roachpb.MakeTenantID(10))
			tenant.Exec(`SET CLUSTER SETTING sql.zone_configs.allow_for_secondary_tenant.enabled = true`)
		} else {
			tenant = spanConfigTestCluster.InitializeTenant(ctx, roachpb.SystemTenantID)
		}

		datadriven.RunTest(t, path, func(t *testing.T, d *datadriven.TestData) string {
			switch d.Cmd {
			case "exec-sql":
				tenant.Exec(d.Input)

			case "query-sql":
				rows := tenant.Query(d.Input)
				output, err := sqlutils.RowsToDataDrivenOutput(rows)
				require.NoError(t, err)
				return output

			case "translate":
				// Parse the args to get the object ID we're looking to
				// translate.
				var objID descpb.ID
				switch {
				case d.HasArg("named-zone"):
					var zone string
					d.ScanArgs(t, "named-zone", &zone)
					namedZoneID, found := zonepb.NamedZones[zonepb.NamedZone(zone)]
					require.Truef(t, found, "unknown named zone: %s", zone)
					objID = descpb.ID(namedZoneID)
				case d.HasArg("id"):
					var scanID int
					d.ScanArgs(t, "id", &scanID)
					objID = descpb.ID(scanID)
				case d.HasArg("database"):
					var dbName string
					d.ScanArgs(t, "database", &dbName)
					if d.HasArg("table") {
						var tbName string
						d.ScanArgs(t, "table", &tbName)
						objID = tenant.LookupTableByName(ctx, dbName, tbName).GetID()
					} else {
						objID = tenant.LookupDatabaseByName(ctx, dbName).GetID()
					}
				default:
					d.Fatalf(t, "insufficient/improper args (%v) provided to translate", d.CmdArgs)
				}

				sqlTranslator := tenant.SpanConfigSQLTranslator().(spanconfig.SQLTranslator)
				entries, _, err := sqlTranslator.Translate(ctx, descpb.IDs{objID})
				require.NoError(t, err)
				sort.Slice(entries, func(i, j int) bool {
					return entries[i].Span.Key.Compare(entries[j].Span.Key) < 0
				})

				var output strings.Builder
				for _, entry := range entries {
					output.WriteString(fmt.Sprintf("%-42s %s\n", entry.Span,
						spanconfigtestutils.PrintSpanConfigDiffedAgainstDefaults(entry.Config)))
				}
				return output.String()

			case "full-translate":
				sqlTranslator := tenant.SpanConfigSQLTranslator().(spanconfig.SQLTranslator)
				entries, _, err := spanconfig.FullTranslate(ctx, sqlTranslator)
				require.NoError(t, err)

				sort.Slice(entries, func(i, j int) bool {
					return entries[i].Span.Key.Compare(entries[j].Span.Key) < 0
				})
				var output strings.Builder
				for _, entry := range entries {
					output.WriteString(fmt.Sprintf("%-42s %s\n", entry.Span,
						spanconfigtestutils.PrintSpanConfigDiffedAgainstDefaults(entry.Config)))
				}
				return output.String()

			case "mark-table-offline":
				var dbName, tbName string
				d.ScanArgs(t, "database", &dbName)
				d.ScanArgs(t, "table", &tbName)
				tenant.WithMutableTableDescriptor(ctx, dbName, tbName, func(mutable *tabledesc.Mutable) {
					mutable.SetOffline("for testing")
				})

			case "mark-table-public":
				var dbName, tbName string
				d.ScanArgs(t, "database", &dbName)
				d.ScanArgs(t, "table", &tbName)
				tenant.WithMutableTableDescriptor(ctx, dbName, tbName, func(mutable *tabledesc.Mutable) {
					mutable.SetPublic()
				})

			default:
				t.Fatalf("unknown command: %s", d.Cmd)
			}

			return ""
		})
	})
}
