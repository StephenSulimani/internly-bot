package models

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Job struct {
	gorm.Model
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	SourceURL       string    `json:"sourceURL"`
	Source          string    `json:"source"`
	JobType         JobType   `json:"jobType"`
	Company         string    `json:"company"`
	Logo            string    `json:"logo"`
	Role            string    `json:"role"`
	Location        string    `json:"location"`
	ApplicationLink string    `json:"application" gorm:"unique"`
	FirstSeen       time.Time `json:"firstSeen"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
}

func (j *Job) BeforeCreate(tx *gorm.DB) (err error) {
	j.ID = uuid.New()
	return
}

type ClearBitResponse struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
	Logo   string `json:"logo"`
}

func (j *Job) SourceLogo(db *gorm.DB) (job *Job, err error) {
	client := &http.Client{}
	url := fmt.Sprintf("https://autocomplete.clearbit.com/v1/companies/suggest?query=%s", url.QueryEscape(j.Company))

	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64; rv:139.0) Gecko/20100101 Firefox/139.0",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var response []ClearBitResponse

	err = json.Unmarshal(body, &response)

	if err != nil {
		return nil, err
	}

	if len(response) > 0 {
		j.Logo = response[0].Logo
	}

	if db != nil {
		return j, db.Save(j).Error
	}

	return j, nil

}
