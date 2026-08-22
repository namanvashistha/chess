package dto

// CreatedUser is the response to POST /api/user. It is the only place the
// session token crosses the wire outbound: the client stores it and sends it
// back to authenticate. dao.User itself hides the token (see dao/user.go).
type CreatedUser struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Status   int    `json:"status"`
	MetaData string `json:"meta_data"`
	Token    string `json:"token"`
}

// UpdateUserRequest is the accepted body for PUT /api/user/:userID.
type UpdateUserRequest struct {
	Name string `json:"name"`
}
