package main

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/ndane/parcelwatch/fetcher"
	"github.com/ndane/parcelwatch/sms"
)

func main() {
	subdomain, ok := os.LookupEnv("RS_SUBDOMAIN")
	if !ok {
		log.Panic("no RS_SUBDOMAIN env var set")
	}

	parcelChan := fetcher.NewFetcher(subdomain, time.Hour, os.Getenv("RS_USERNAME"), os.Getenv("RS_PASSWORD"))
	toNumber, ok := os.LookupEnv("TO_NUMBER")
	if !ok {
		log.Error("no TO_NUMBER env var set")
	}

	sender := setupTwilioSender()

	knownParcels := make([]fetcher.Parcel, 0)

	for {
		parcels := <-parcelChan
		delta := len(parcels) - len(knownParcels)
		if delta > 0 {
			fmt.Println(fmt.Sprintf("%d new parcels detected", delta))

			newParcels := make([]fetcher.Parcel, 0)
			for i := 0; i < delta; i++ {
				p := parcels[i]
				newParcels = append(newParcels, p)

				fmt.Println(fmt.Sprintf("Code: %s\tCollected By: %s\tCollected Date: %s",
					p.Code,
					p.CollectedBy,
					p.CollectedDate.Format(time.RFC850)))
			}

			if len(knownParcels) > 0 {
				if err := sender.Send(toNumber, fmt.Sprintf("%d new parcels delivered", delta)); err != nil {
					log.WithError(err).Error("failed to send delivery notification sms")
				}
			}

			knownParcels = parcels
		}
	}
}

func setupTwilioSender() sms.Sender {
	twilioToken, ok := os.LookupEnv("TW_TOKEN")
	if !ok {
		log.Error("no TW_TOKEN env var set")
	}

	twilioSid, ok := os.LookupEnv("TW_SID")
	if !ok {
		log.Error("no TW_SID env var set")
	}

	twilioNumber, ok := os.LookupEnv("TW_NUMBER")
	if !ok {
		log.Error("no TW_NUMBER env var set")
	}

	return sms.NewTwilioSender(twilioToken, twilioSid, twilioNumber)
}
