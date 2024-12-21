package dao

type User struct {
	ID       int    `gorm:"column:id; primary_key; not null" json:"id"`
	Name     string `gorm:"column:name" json:"name"`
	Token    string `gorm:"column:token;uniqueIndex" json:"token"`
	Status   int    `gorm:"column:status" json:"status"`
	MetaData string `gorm:"column:meta_data" json:"meta_data"`
	BaseModel
}
