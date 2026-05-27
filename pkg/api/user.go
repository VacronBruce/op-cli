package api

// User represents an OpenProject user.
type User struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Login  string `json:"login"`
	Email  string `json:"email"`
	Status string `json:"status"`
	Links  struct {
		Self Link `json:"self"`
	} `json:"_links"`
}

// GetMe retrieves the current authenticated user.
func (c *Client) GetMe() (*User, error) {
	var u User
	if err := c.Get("/users/me", &u); err != nil {
		return nil, err
	}
	return &u, nil
}
