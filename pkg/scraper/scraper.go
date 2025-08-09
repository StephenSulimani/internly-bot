package scraper

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/stephensulimani/internly-bot/pkg/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Scraper interface {
	Scrape() ([]models.Job, error)
}

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

	slices.Reverse(matches)

	for i, match := range matches {
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
			SourceURL:       s.URL,
			Source:          s.Name,
			JobType:         s.JobType,
			Company:         match[companyGroup],
			Role:            match[roleGroup],
			Location:        match[locationGroup],
			ApplicationLink: match[applicationLinkGroup],
			FirstSeen:       time.Now(),
		}

		if s.AgeGroup != 0 {
			// if strings.Contains(match[s.AgeGroup], "mo") {
			// 	match[s.AgeGroup] = strings.ReplaceAll(match[s.AgeGroup], "mo", "M")
			// }
			duration, err := ParseDuration(match[s.AgeGroup])

			if err == nil {
				job.FirstSeen = time.Now().Add(-duration)
			} else {
				fmt.Println(err)
			}
		}

		next := i + 1
		for job.Company == "" {
			job.Company = matches[next][companyGroup]
			next += 1
		}

		re := regexp.MustCompile(`<[^>]*>`)
		cleanedString := re.ReplaceAllString(job.Location, " ")

		cleanedString = strings.ReplaceAll(cleanedString, "*", "")
		cleanedString = strings.TrimSpace(cleanedString)

		job.Location = cleanedString

		err = db.Save(&job).Error

		if err != nil {
			if err == gorm.ErrDuplicatedKey {
				continue
			}
			log.Error(err)
		}

		_, err := job.SourceLogo(db)
		if err != nil {
			log.Error(err)
		}

		if jobEvent != nil {
			*jobEvent <- job
		}

		jobs = append(jobs, job)
	}

	return jobs, nil

}

// ParseDuration parses a duration string.
// examples: "10d", "-1.5w" or "3Y4M5d".
// Add time units are "d"="D", "w"="W", "M", "y"="Y".
func ParseDuration(s string) (time.Duration, error) {
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}

	re := regexp.MustCompile(`(\d*\.\d+|\d+)[^\d]*`)
	unitMap := map[string]time.Duration{
		"d":  24,
		"D":  24,
		"w":  7 * 24,
		"W":  7 * 24,
		"M":  30 * 24,
		"mo": 30 * 24,
		"y":  365 * 24,
		"Y":  365 * 24,
	}

	strs := re.FindAllString(s, -1)
	var sumDur time.Duration
	for _, str := range strs {
		var _hours time.Duration = 1
		for unit, hours := range unitMap {
			if strings.Contains(str, unit) {
				str = strings.ReplaceAll(str, unit, "h")
				_hours = hours
				break
			}
		}

		dur, err := time.ParseDuration(str)
		if err != nil {
			return 0, err
		}

		sumDur += dur * _hours
	}

	if neg {
		sumDur = -sumDur
	}
	return sumDur, nil
}
