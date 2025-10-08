package constants

// polymer response codes
// these consist of 4 digitsnumbers
//
// the 1st 3 are randomly generated but represent specific scenarios
// 4th indicates if the response requires user interactions through a dialog box. 0 means it does not require. 1 means it requires.

var ENCRYPTION_KEY_EXPIRED uint = 6170                   // request a new encryption key
var UNVERIFIED_EMAIL_LOGIN_ATTEMPT uint = 4110           // take the user to the face match page to unlock the account
var ACCOUNT_CREATED uint = 9110                          // take the user to the otp page to verify email or phone
var ACCOUNT_EXISTS uint = 9120                           // take the user to the face match page to register the device
var ACCOUNT_EXISTS_UNVERIFIED uint = 9130                // take the user to the face match page to register the device
var ACCOUNT_EXISTS_EMAIL_OR_PHONE_UNVERIFIED uint = 9140 // take the user to the otp page to verify email or phone
var FREE_TIER_ACCOUNT_LIMIT_HIT uint = 5243              // display a page telling the user the limit has been hit
var SET_APP_PIN uint = 1433                              // display a page telling the user the limit has been hit
var VERIFY_WORKSPACE_MEMBER_EMAIL uint = 1937            // display a page telling the user the limit has been hit

var AVAILABLE_REQUIRED_DATA_POINTS = []string{"BVN", "NIN", "FirstName", "LastName", "Gender", "MiddleName", "DOB", "Image", "Email", "Phone", "LoginLocale"} // include "address" later
var CUSTOM_FIELD_TYPES = []string{"long_text", "short_text", "switch", "dropdown", "number", "secret", "pin", "date"}

var MAX_ORGANISATIONS_CREATED int64 = 20

var SUPPORT_EMAIL = "help@gateman.io"

var FREE_TIER_MAU_LIMIT int64 = 10_000
var PAID_TIER_FREE_MAU_LIMIT int64 = 40_000
var ESSENTIAL_TIER_MAU_PRICE int64 = 20_00
var PREMIUM_TIER_MAU_PRICE int64 = 12_00
