package common

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetEmployeeId(c *gin.Context) (uuid.UUID, error) {
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		return uuid.Nil, errors.New("employee ID missing")
	}

	empIDStr, ok := empIDRaw.(string)
	if !ok {
		return uuid.Nil, errors.New("invalid employee ID format")
	}

	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		return uuid.Nil, errors.New("invalid employee UUID")
	}

	return empID, nil
}

func GetMonthYear(c *gin.Context) (int, int, error) {
	month, err := strconv.Atoi(
		c.DefaultQuery("month", fmt.Sprintf("%d", int(time.Now().Month()))),
	)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month")
	}

	year, err := strconv.Atoi(
		c.DefaultQuery("year", fmt.Sprintf("%d", time.Now().Year())),
	)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year")
	}

	return month, year, nil
}
