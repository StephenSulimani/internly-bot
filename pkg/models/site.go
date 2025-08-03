package models

type JobType string

const (
	NEW_GRAD JobType = "NEW_GRAD"
	INTERN   JobType = "INTERN"
)

type Site struct {
	URL                  string  `json:"string"`
	RegexPattern         string  `json:"regexPattern"`
	JobType              JobType `json:"type"`
	CompanyGroup         int     `json:"companyGroup"`
	RoleGroup            int     `json:"roleGroup"`
	LocationGroup        int     `json:"locationGroup"`
	ApplicationLinkGroup int     `json:"applicationLinkGroup"`
}
