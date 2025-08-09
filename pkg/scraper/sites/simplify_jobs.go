package sites

import (
	"encoding/json"
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
	urls := []string{"https://raw.githubusercontent.com/SimplifyJobs/Summer2026-Internships/refs/heads/dev/.github/scripts/listings.json", "https://raw.githubusercontent.com/SimplifyJobs/New-Grad-Positions/refs/heads/dev/.github/scripts/listings.json"}

	jobs := []models.Job{}

	for _, url := range urls {
		source := "Simplify.jobs"
		sj.log.Infof("Starting Scrape: %s", url)
		client := &http.Client{}

		req, err := http.NewRequest(http.MethodGet, url, nil)

		if err != nil {
			return jobs, err
		}

		resp, err := client.Do(req)

		if err != nil {
			return jobs, err
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			return jobs, err
		}

		simplifyJobs := []simplifyJob{}

		err = json.Unmarshal(body, &simplifyJobs)

		if err != nil {
			return jobs, err
		}

		localJobs := []models.Job{}

		for _, job := range simplifyJobs {
			firstSeen := time.Unix(int64(job.DateUpdated), 0)
			if firstSeen.Unix() < time.Now().Add(-35*24*time.Hour).Unix() {
				continue
			}
			jobType := models.NEW_GRAD

			if strings.Contains(url, "Intern") {
				jobType = models.INTERN
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

		for _, job := range localJobs {
			err := sj.db.Save(&job).Error

			if err != nil {
				if err == gorm.ErrDuplicatedKey {
					continue
				}
				sj.log.Error(err)
				continue
			}

			job.SourceLogo(sj.db)

			if sj.jobChan != nil {
				*sj.jobChan <- job
			}
		}

		jobs = append(jobs, localJobs...)

		sj.log.Infof("Finished Scrape: %s | %d jobs", url, len(localJobs))

	}

	return jobs, nil
}
