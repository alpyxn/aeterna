package models

// ApplicationSettings holds global singleton configuration (single row, id = 1).
type ApplicationSettings struct {
	ID                uint `gorm:"primaryKey"`
	AllowRegistration bool `gorm:"column:allow_registration;default:0" json:"allow_registration"`
}
