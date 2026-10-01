package sites

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/stephensulimani/internly-bot/pkg/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type simplifyJob struct {
	Company         string   `json:"company_name"`
	Locations       []string `json:"locations"`
	Role            string   `json:"title"`
	ApplicationLink string   `json:"url"`
	DatePosted      int      `json:"date_posted"`
	DateUpdated     int      `json:"date_updated"`
	Active          bool     `json:"active"`
}

type simplifyJobs struct {
	log     *zap.SugaredLogger
	db      *gorm.DB
	jobChan *chan models.Job
}

func NewSimplifyJobs(log *zap.SugaredLogger, db *gorm.DB, jobChan *chan models.Job) *simplifyJobs {
	return &simplifyJobs{
		log:     log,
		db:      db,
		jobChan: jobChan,
	}
}

func (sj *simplifyJobs) Scrape() ([]models.Job, error) {
	urls := []string{
		"https://raw.githubusercontent.com/SimplifyJobs/Summer2026-Internships/refs/heads/dev/.github/scripts/listings.json",
		"https://raw.githubusercontent.com/SimplifyJobs/New-Grad-Positions/refs/heads/dev/.github/scripts/listings.json",
	}

	client := &http.Client{Timeout: 60 * time.Second}
	jobs := []models.Job{}

	for _, url := range urls {
		source := "Simplify.jobs"
		sj.log.Infof("Starting Scrape: %s", url)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return jobs, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return jobs, err
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		resp.Body.Close()
		if err != nil {
			return jobs, err
		}
		if resp.StatusCode != http.StatusOK {
			return jobs, fmt.Errorf("scrape %s: unexpected status %d", url, resp.StatusCode)
		}

		var listings []simplifyJob
		if err := json.Unmarshal(body, &listings); err != nil {
			return jobs, err
		}

		jobType := models.NEW_GRAD
		if strings.Contains(url, "Intern") {
			jobType = models.INTERN
		}

		cutoff := time.Now().Add(-35 * 24 * time.Hour)
		localJobs := make([]models.Job, 0, len(listings))

		for _, job := range listings {
			if !job.Active {
				continue
			}
			firstSeen := time.Unix(int64(job.DateUpdated), 0)
			if firstSeen.Before(cutoff) {
				continue
			}
			localJobs = append(localJobs, models.Job{
				Company:         job.Company,
				Location:        strings.Join(job.Locations, ", "),
				Role:            job.Role,
				JobType:         jobType,
				ApplicationLink: job.ApplicationLink,
				FirstSeen:       firstSeen,
				Source:          source,
				SourceURL:       url,
			})
		}

		saved := 0
		for i := range localJobs {
			job := &localJobs[i]
			err := sj.db.Create(job).Error
			if err != nil {
				if err == gorm.ErrDuplicatedKey {
					continue
				}
				sj.log.Error(err)
				continue
			}
			saved++

			if sj.jobChan != nil {
				*sj.jobChan <- *job
			}
		}

		jobs = append(jobs, localJobs...)
		sj.log.Infof("Finished Scrape: %s | %d active recent jobs (%d newly saved)", url, len(localJobs), saved)
	}

	return jobs, nil
}
