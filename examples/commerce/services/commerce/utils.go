package commerce

import (
	"fmt"
	"strconv"
)

func formatMoney(value any) string {
	var cents int64
	switch number := value.(type) {
	case int64:
		cents = number
	case int32:
		cents = int64(number)
	case int:
		cents = int64(number)
	case string:
		cents, _ = strconv.ParseInt(number, 10, 64)
	case []byte:
		cents, _ = strconv.ParseInt(string(number), 10, 64)
	}
	return fmt.Sprintf("$%d.%02d", cents/100, cents%100)
}
