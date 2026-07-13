package api

import "time"

type Household struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	GuildID   string     `json:"guildId"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type ResolveHouseholdQuery struct {
	Name    *string
	GuildID *string
}
