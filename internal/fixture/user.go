package fixture

import "time"

// User represents the basic user data.
type User struct {
	Username      string
	Age           int
	Height        float32
	Email         string
	VerifiedEmail bool
	Address       Address
	CreationTime  time.Time
}

// User1 is an example of user that can be used for testing.
var User1 = User{
	Username:      "johndoe",
	Age:           24,
	Height:        1.82,
	Email:         "john.doe@gmail.com",
	VerifiedEmail: false,
	Address:       Address1,
	CreationTime:  time.Date(2024, time.August, 24, 10, 25, 59, 44, time.UTC),
}

// User2 is an example of user that can be used for testing.
var User2 = User{
	Username:      "janedoe",
	Age:           32,
	Height:        1.68,
	Email:         "jane.doe@gmail.com",
	VerifiedEmail: true,
	Address:       Address2,
	CreationTime:  time.Date(2024, time.August, 20, 8, 43, 26, 18, time.UTC),
}
