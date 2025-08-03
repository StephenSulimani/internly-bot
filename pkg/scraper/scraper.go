package scraper

import (
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/stephensulimani/internly-bot/pkg/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Scrape first scrapes the site and parses the page for relevant information.
// It then creates a new Job and attempts to add it to the database.
// If there is a UNIQUE constraint violation, the job is skipped.
// Otherwise, it is saved, the job is sent through the jobEvent channel, and it is added to the jobs slice.
// Finally, the jobs slice is returned, containing only new jobs.
func Scrape(s *models.Site, db *gorm.DB, jobEvent *chan models.Job, log *zap.SugaredLogger) ([]models.Job, error) {
	log.Infof("Starting Scrape: %s", s.URL)
	defer log.Infof("Finished Scrape: %s", s.URL)

	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, s.URL, nil)

	if err != nil {
		log.Error(err)
		return nil, err
	}

	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
		"Accept":          "*/*",
		"Accept-Language": "en-US,en;q=0.9",
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)

	if err != nil {
		log.Error(err)
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Errorf("Received non-200 status code: %d", resp.StatusCode)
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Error(err)
		return nil, err
	}

	regex := regexp.MustCompile(s.RegexPattern)

	matches := regex.FindAllStringSubmatch(string(body), -1)

	jobs := []models.Job{}

	for _, match := range matches {
		companyGroup := 1
		roleGroup := 2
		locationGroup := 3
		applicationLinkGroup := 4

		if s.CompanyGroup != 0 {
			companyGroup = s.CompanyGroup
		}

		if s.RoleGroup != 0 {
			roleGroup = s.RoleGroup
		}

		if s.LocationGroup != 0 {
			locationGroup = s.LocationGroup
		}

		if s.ApplicationLinkGroup != 0 {
			applicationLinkGroup = s.ApplicationLinkGroup
		}

		job := models.Job{
			Source:          s.URL,
			JobType:         s.JobType,
			Company:         match[companyGroup],
			Role:            match[roleGroup],
			Location:        match[locationGroup],
			ApplicationLink: match[applicationLinkGroup],
			FirstSeen:       time.Now(),
		}

		err = db.Save(&job).Error

		if err != nil {
			if err == gorm.ErrDuplicatedKey {
				log.Infof("Job already exists: %s", job.ApplicationLink)
				continue
			}
		}

		if jobEvent != nil {
			*jobEvent <- job
		}

		jobs = append(jobs, job)
	}

	return jobs, nil

}
