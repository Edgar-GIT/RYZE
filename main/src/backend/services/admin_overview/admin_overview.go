// Package admin_overview builds the two administration dashboards: the
// business overview of the Management Administrator (sales, members,
// training activity, catalogue and work queues) and the platform overview of
// the Technical Administrator (accounts, activity and admin audit health).
// Every figure is an aggregate; no personal or payment data is returned.
package admin_overview

import (
	"context"
	"time"

	"ryze/backend/repositories"
)

// AllowedRanges are the dashboard periods in days.
var AllowedRanges = []int{7, 30, 90}

// DefaultRange is used when the requested range is not allowed.
const DefaultRange = 30

// DayPoint is one day of a series; MinorUnits is set for revenue series.
type DayPoint struct {
	Date       string
	Count      int64
	MinorUnits int64
}

// Share is one labelled aggregate.
type Share struct {
	Label      string
	Count      int64
	MinorUnits int64
}

// RankedProgram is one programme of a ranking.
type RankedProgram struct {
	ID         string
	Name       string
	Family     string
	Count      int64
	MinorUnits int64
}

// Business is the Management Administrator overview.
type Business struct {
	GeneratedAt         time.Time
	RangeDays           int
	RevenueAllTime      int64
	SalesAllTime        int64
	Revenue             int64
	PreviousRevenue     int64
	Sales               int64
	PreviousSales       int64
	PlatformShare       int64
	TrainerShare        int64
	RevenueByFamily     []Share
	RevenueDaily        []DayPoint
	PendingPurchases    int64
	FailedPurchases     int64
	TestPurchases       int64
	Members             int64
	NewMembers          int64
	SignupsDaily        []DayPoint
	ActiveMembers7d     int64
	ActiveMembersRange  int64
	SessionsDaily       []DayPoint
	SessionsCompleted   int64
	FreeActivations     int64
	ActiveEntitlements  int64
	ActiveTrainers      int64
	PendingApplications int64
	PublishedByFamily   []Share
	PlatformDrafts      int64
	Queues              map[string]int64
	TopSelling          []RankedProgram
	TopViewed           []RankedProgram
}

// Platform is the Technical Administrator overview.
type Platform struct {
	GeneratedAt         time.Time
	RangeDays           int
	Members             int64
	NewMembers          int64
	VerifiedMembers     int64
	DisabledUsers       int64
	SignupsDaily        []DayPoint
	ActiveMembers7d     int64
	SessionsDaily       []DayPoint
	SessionsCompleted   int64
	ActiveTrainers      int64
	PendingApplications int64
	AuditEvents24h      int64
	AuditDenied24h      int64
	AuditFailed24h      int64
}

// Service builds the overviews.
type Service interface {
	Business(ctx context.Context, rangeDays int) (*Business, error)
	Platform(ctx context.Context, rangeDays int) (*Platform, error)
	// Queues returns the open work counts shown as navigation badges.
	Queues(ctx context.Context) (map[string]int64, error)
}

type service struct {
	repo repositories.AdminOverviewRepository
	now  func() time.Time
}

func NewService(repo repositories.AdminOverviewRepository) Service {
	return &service{repo: repo, now: time.Now}
}

// NormalizeRange maps any requested range to an allowed one.
func NormalizeRange(days int) int {
	for _, allowed := range AllowedRanges {
		if days == allowed {
			return days
		}
	}
	return DefaultRange
}

// rangeStart returns the start of the first UTC day of a range of n days that
// ends today.
func rangeStart(now time.Time, days int) time.Time {
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	return today.AddDate(0, 0, -(days - 1))
}

// fillDays turns sparse daily rows into a complete series with zero days.
func fillDays(from time.Time, days int, rows []repositories.DayCount) []DayPoint {
	byDay := make(map[string]repositories.DayCount, len(rows))
	for _, row := range rows {
		byDay[row.Day.UTC().Format("2006-01-02")] = row
	}
	series := make([]DayPoint, 0, days)
	for i := 0; i < days; i++ {
		key := from.AddDate(0, 0, i).Format("2006-01-02")
		row := byDay[key]
		series = append(series, DayPoint{Date: key, Count: row.Count, MinorUnits: row.MinorUnits})
	}
	return series
}

func shares(rows []repositories.LabelCount) []Share {
	result := make([]Share, 0, len(rows))
	for _, row := range rows {
		result = append(result, Share{Label: row.Label, Count: row.Count, MinorUnits: row.MinorUnits})
	}
	return result
}

func ranked(rows []repositories.TopProgram) []RankedProgram {
	result := make([]RankedProgram, 0, len(rows))
	for _, row := range rows {
		result = append(result, RankedProgram{ID: row.ID, Name: row.Name, Family: row.Family, Count: row.Count, MinorUnits: row.MinorUnits})
	}
	return result
}

func sum(points []DayPoint) int64 {
	var total int64
	for _, point := range points {
		total += point.Count
	}
	return total
}

func (s *service) Business(ctx context.Context, rangeDays int) (*Business, error) {
	days := NormalizeRange(rangeDays)
	now := s.now()
	from := rangeStart(now, days)
	previousFrom := from.AddDate(0, 0, -days)
	overview := &Business{GeneratedAt: now.UTC(), RangeDays: days}

	allTime, err := s.repo.RevenueAllTime(ctx)
	if err != nil {
		return nil, err
	}
	overview.RevenueAllTime, overview.SalesAllTime = allTime.MinorUnits, allTime.Count

	current, err := s.repo.Revenue(ctx, from, now.Add(time.Minute))
	if err != nil {
		return nil, err
	}
	overview.Revenue, overview.Sales = current.MinorUnits, current.Count
	overview.PlatformShare, overview.TrainerShare = current.PlatformAmount, current.TrainerAmount

	previous, err := s.repo.Revenue(ctx, previousFrom, from)
	if err != nil {
		return nil, err
	}
	overview.PreviousRevenue, overview.PreviousSales = previous.MinorUnits, previous.Count

	byFamily, err := s.repo.RevenueByFamily(ctx, from)
	if err != nil {
		return nil, err
	}
	overview.RevenueByFamily = shares(byFamily)

	daily, err := s.repo.RevenueDaily(ctx, from)
	if err != nil {
		return nil, err
	}
	overview.RevenueDaily = fillDays(from, days, daily)

	if overview.PendingPurchases, overview.FailedPurchases, overview.TestPurchases, err = s.repo.PurchaseCounts(ctx, from); err != nil {
		return nil, err
	}
	if overview.Members, overview.NewMembers, _, err = s.repo.MemberCounts(ctx, from); err != nil {
		return nil, err
	}
	signups, err := s.repo.SignupsDaily(ctx, from)
	if err != nil {
		return nil, err
	}
	overview.SignupsDaily = fillDays(from, days, signups)

	if overview.ActiveMembers7d, err = s.repo.ActiveMembers(ctx, now.AddDate(0, 0, -7)); err != nil {
		return nil, err
	}
	if overview.ActiveMembersRange, err = s.repo.ActiveMembers(ctx, from); err != nil {
		return nil, err
	}
	sessions, err := s.repo.SessionsCompletedDaily(ctx, from)
	if err != nil {
		return nil, err
	}
	overview.SessionsDaily = fillDays(from, days, sessions)
	overview.SessionsCompleted = sum(overview.SessionsDaily)

	if overview.FreeActivations, err = s.repo.FreeActivations(ctx, from); err != nil {
		return nil, err
	}
	if overview.ActiveEntitlements, err = s.repo.ActiveEntitlements(ctx); err != nil {
		return nil, err
	}
	if overview.ActiveTrainers, overview.PendingApplications, err = s.repo.TrainerCounts(ctx); err != nil {
		return nil, err
	}
	published, drafts, err := s.repo.CatalogueCounts(ctx)
	if err != nil {
		return nil, err
	}
	overview.PublishedByFamily, overview.PlatformDrafts = shares(published), drafts

	if overview.Queues, err = s.repo.QueueCounts(ctx); err != nil {
		return nil, err
	}
	topSelling, err := s.repo.TopSellingPrograms(ctx, from, 5)
	if err != nil {
		return nil, err
	}
	overview.TopSelling = ranked(topSelling)
	topViewed, err := s.repo.TopViewedPrograms(ctx, 5)
	if err != nil {
		return nil, err
	}
	overview.TopViewed = ranked(topViewed)
	return overview, nil
}

func (s *service) Platform(ctx context.Context, rangeDays int) (*Platform, error) {
	days := NormalizeRange(rangeDays)
	now := s.now()
	from := rangeStart(now, days)
	overview := &Platform{GeneratedAt: now.UTC(), RangeDays: days}

	var err error
	if overview.Members, overview.NewMembers, overview.VerifiedMembers, err = s.repo.MemberCounts(ctx, from); err != nil {
		return nil, err
	}
	if overview.DisabledUsers, err = s.repo.DisabledUsers(ctx); err != nil {
		return nil, err
	}
	signups, err := s.repo.SignupsDaily(ctx, from)
	if err != nil {
		return nil, err
	}
	overview.SignupsDaily = fillDays(from, days, signups)
	if overview.ActiveMembers7d, err = s.repo.ActiveMembers(ctx, now.AddDate(0, 0, -7)); err != nil {
		return nil, err
	}
	sessions, err := s.repo.SessionsCompletedDaily(ctx, from)
	if err != nil {
		return nil, err
	}
	overview.SessionsDaily = fillDays(from, days, sessions)
	overview.SessionsCompleted = sum(overview.SessionsDaily)
	if overview.ActiveTrainers, overview.PendingApplications, err = s.repo.TrainerCounts(ctx); err != nil {
		return nil, err
	}
	if overview.AuditEvents24h, overview.AuditDenied24h, overview.AuditFailed24h, err = s.repo.AuditCounts(ctx, now.Add(-24*time.Hour)); err != nil {
		return nil, err
	}
	return overview, nil
}

func (s *service) Queues(ctx context.Context) (map[string]int64, error) {
	queues, err := s.repo.QueueCounts(ctx)
	if err != nil {
		return nil, err
	}
	_, pending, err := s.repo.TrainerCounts(ctx)
	if err != nil {
		return nil, err
	}
	queues["applications_pending"] = pending
	return queues, nil
}
