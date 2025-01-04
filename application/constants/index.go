package constants

// polymer response codes
// these consist of 4 digitsnumbers
//
// the 1st 3 are randomly generated but represent specific scenarios
// 4th indicates if the response requires user interactions through a dialog box. 0 means it does not require. 1 means it requires.

var ENCRYPTION_KEY_EXPIRED uint = 6170                   // request a new encryption key
var UNVERIFIED_EMAIL_LOGIN_ATTEMPT uint = 4110           // take the user to the face match page to unlock the account
var ACCOUNT_CREATED uint = 9110                          // take the user to the face match page to register the device
var ACCOUNT_EXISTS uint = 9120                           // take the user to the face match page to register the device
var ACCOUNT_EXISTS_UNVERIFIED uint = 9130                // take the user to the face match page to register the device
var ACCOUNT_EXISTS_EMAIL_OR_PHONE_UNVERIFIED uint = 9140 // take the user to the verify otp page to verify email

var AVAILABLE_REQUIRED_DATA_POINTS = []string{"bvn", "nin", "address", "biometric", "email", "phone", "login_location"}

var MAX_ORGANISATIONS_CREATED int64 = 20

var SUPPORT_EMAIL = "help@gateman.io"