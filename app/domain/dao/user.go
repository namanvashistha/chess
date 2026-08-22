package dao

type User struct {
	ID   int    `gorm:"column:id; primary_key; not null" json:"id"`
	Name string `gorm:"column:name" json:"name"`
	// Token is a session credential. It is deliberately NOT serialised: this
	// struct is embedded in ChessGame.WhiteUser/BlackUser, and that whole game
	// object is broadcast to every client watching the game. With a `json:"token"`
	// tag, any cache-hit broadcast handed both players' tokens to each other.
	// The token is returned exactly once, by AddUserData, via dto.CreatedUser.
	Token    string `gorm:"column:token;uniqueIndex" json:"-"`
	Status   int    `gorm:"column:status" json:"status"`
	MetaData string `gorm:"column:meta_data" json:"meta_data"`
	BaseModel
}
