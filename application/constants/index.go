package constants

// polymer response codes
// these consist of 4 digitsnumbers
//
// the 1st 3 are randomly generated but represent specific scenarios
// 4th indicates if the response requires user interactions through a dialog box. 0 means it does not require. 1 means it requires.

var ENCRYPTION_KEY_EXPIRED uint = 6170 // take the user to the face match page to unlock the account
