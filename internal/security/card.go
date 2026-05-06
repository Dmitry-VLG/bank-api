package security

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

type CardData struct {
	Number string
	Expiry string
	CVV    string
	Last4  string
}

func GenerateCard() (CardData, error) {
	prefix := "400000"

	body, err := randomDigits(9)
	if err != nil {
		return CardData{}, err
	}

	withoutCheck := prefix + body
	check := luhnCheckDigit(withoutCheck)
	number := fmt.Sprintf("%s%d", withoutCheck, check)

	cvv, err := randomDigits(3)
	if err != nil {
		return CardData{}, err
	}

	expiry := time.Now().AddDate(3, 0, 0).Format("01/06")

	return CardData{
		Number: number,
		Expiry: expiry,
		CVV:    cvv,
		Last4:  number[len(number)-4:],
	}, nil
}

func ValidLuhn(number string) bool {
	sum := 0
	double := false

	for i := len(number) - 1; i >= 0; i-- {
		d := int(number[i] - '0')
		if d < 0 || d > 9 {
			return false
		}

		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}

		sum += d
		double = !double
	}

	return sum%10 == 0
}

func luhnCheckDigit(number string) int {
	sum := 0
	double := true

	for i := len(number) - 1; i >= 0; i-- {
		d := int(number[i] - '0')

		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}

		sum += d
		double = !double
	}

	return (10 - sum%10) % 10
}

func randomDigits(n int) (string, error) {
	out := make([]byte, n)

	for i := range out {
		v, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}

		out[i] = byte('0' + v.Int64())
	}

	return string(out), nil
}
