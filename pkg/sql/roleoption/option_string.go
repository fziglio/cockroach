// Code generated by "stringer"; DO NOT EDIT.

package roleoption

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[CREATEROLE-1]
	_ = x[NOCREATEROLE-2]
	_ = x[PASSWORD-3]
	_ = x[LOGIN-4]
	_ = x[NOLOGIN-5]
	_ = x[VALIDUNTIL-6]
	_ = x[CONTROLJOB-7]
	_ = x[NOCONTROLJOB-8]
	_ = x[CONTROLCHANGEFEED-9]
	_ = x[NOCONTROLCHANGEFEED-10]
	_ = x[CREATEDB-11]
	_ = x[NOCREATEDB-12]
	_ = x[CREATELOGIN-13]
	_ = x[NOCREATELOGIN-14]
	_ = x[VIEWACTIVITY-15]
	_ = x[NOVIEWACTIVITY-16]
	_ = x[CANCELQUERY-17]
	_ = x[NOCANCELQUERY-18]
	_ = x[MODIFYCLUSTERSETTING-19]
	_ = x[NOMODIFYCLUSTERSETTING-20]
	_ = x[DEFAULTSETTINGS-21]
	_ = x[VIEWACTIVITYREDACTED-22]
	_ = x[NOVIEWACTIVITYREDACTED-23]
	_ = x[SQLLOGIN-24]
	_ = x[NOSQLLOGIN-25]
}

const _Option_name = "CREATEROLENOCREATEROLEPASSWORDLOGINNOLOGINVALIDUNTILCONTROLJOBNOCONTROLJOBCONTROLCHANGEFEEDNOCONTROLCHANGEFEEDCREATEDBNOCREATEDBCREATELOGINNOCREATELOGINVIEWACTIVITYNOVIEWACTIVITYCANCELQUERYNOCANCELQUERYMODIFYCLUSTERSETTINGNOMODIFYCLUSTERSETTINGDEFAULTSETTINGSVIEWACTIVITYREDACTEDNOVIEWACTIVITYREDACTEDSQLLOGINNOSQLLOGIN"

var _Option_index = [...]uint16{0, 10, 22, 30, 35, 42, 52, 62, 74, 91, 110, 118, 128, 139, 152, 164, 178, 189, 202, 222, 244, 259, 279, 301, 309, 319}

func (i Option) String() string {
	i -= 1
	if i >= Option(len(_Option_index)-1) {
		return "Option(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _Option_name[_Option_index[i]:_Option_index[i+1]]
}
