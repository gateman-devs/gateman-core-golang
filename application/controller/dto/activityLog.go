package dto

type FetchActivityLogsDTO struct {
	AppID      string  `json:"appID" validate:"required"`
	IPAddress  *string `json:"ipAddress"`
	Method     *string `json:"method"`
	URL        *string `json:"url"`
	StartTime  *string `json:"startTime"`
	EndTime    *string `json:"endTime"`
	PageSize   *int64  `json:"pageSize"`
	LastID     *string `json:"lastID"`
	SortOrder  *int    `json:"sortOrder"` // 1 for ascending, -1 for descending
}
