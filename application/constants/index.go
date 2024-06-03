package constants

// polymer response codes
// these consist of 4 digitsnumbers
//
// the 1st 3 are randomly generated but represent specific scenarios
// 4th indicates if the response requires user interactions through a dialog box. 0 means it does not require. 1 means it requires.

var ENCRYPTION_KEY_EXPIRED uint = 6170 // request a new encryption key
var UNVERIFIED_EMAIL_LOGIN_ATTEMPT uint = 4110 // take the user to the face match page to unlock the account

var AVAILABLE_REQUIRED_DATA_POINTS = []string{"bvn", "nin", "address", "biometric", "email", "phone", "login_location"}

var MAX_LOGIN_ATTEMPTS = 5