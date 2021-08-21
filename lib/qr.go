package lib

import (
	"github.com/yeqown/go-qrcode"
)

func GenerateQR(id string, filename string) (string, error) {
	config := qrcode.Config{
		EncMode: qrcode.EncModeByte,
		EcLevel: qrcode.ErrorCorrectionQuart,
	}

	qr, err := qrcode.NewWithConfig(
		// TODO: use the proper url
		"http://localhost:3000/content/" + filename,
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
