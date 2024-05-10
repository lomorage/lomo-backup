package main

import (
	"os"
	"time"

	"github.com/lomorage/lomo-backup/common/gcloud"
	"github.com/urfave/cli"
)

func gcloudAuth(ctx *cli.Context) error {
	return gcloud.AuthHelper(ctx.String("redirect-path"), ctx.Int("redirect-port"),
		&gcloud.Config{CredFilename: ctx.String("cred"), TokenFilename: ctx.String("token")})
}

func gcloudTokenRefresh(ctx *cli.Context) error {
	tokenFilename := ctx.String("token")
	token, err := gcloud.RefreshAccessToken(&gcloud.Config{
		CredFilename:  ctx.String("cred"),
		TokenFilename: tokenFilename,
	})
	if err != nil {
		return err
	}

	// move current token file to old one, and regenerate new one
	err = os.Rename(tokenFilename, tokenFilename+"."+time.Now().Format("2006-01-02"))
	if err != nil {
		return err
	}

	return gcloud.SaveToken(tokenFilename, token)
}
