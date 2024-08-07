package fixture

// Address represents the address of a user.
type Address struct {
	Street     string
	Number     int
	City       string
	State      string
	PostalCode int
	Country    string
}

// Address1 is an example of address that can be used for testing.
var Address1 = Address{
	Street:     "Apple Park Way",
	Number:     1,
	City:       "Cubertino",
	State:      "California",
	PostalCode: 95014,
	Country:    "United States",
}

// Address2 is an example of address that can be used for testing.
var Address2 = Address{
	Street:     "Amphitheatre Parkway",
	Number:     1600,
	City:       "Mountain View",
	State:      "California",
	PostalCode: 94043,
	Country:    "Unites States",
}
