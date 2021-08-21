package lib

import (
	"github.com/yeqown/go-qrcode"
)

func GenerateQR(id string, filename string) (string, error) {
	appConfig := GetConfig()
	config := qrcode.Config{
		EncMode: qrcode.EncModeByte,
		EcLevel: qrcode.ErrorCorrectionQuart,
	}

	qr, err := qrcode.NewWithConfig(
		appConfig.APP_URL + "/content/" + filename,
		&config,
		qrcode.WithQRWidth(10),
	)

	if err != nil {
		return "", err
	}

	dest := "tmp/" + id + "_qr.png"
	err = qr.Save(dest)

	if err != nil {
		return "", err
	}

	return dest, nil
}
