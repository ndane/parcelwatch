# Resi-Sense Parcel Watcher

Monitor parcels and deliveries on the Resi-Sense platform and get updates via SMS.

## Running the software

### Required environment variables

#### *RS_SUBDOMAIN*

The subdomain of your building at resi-sense.co.uk

#### *RS_USERNAME*

The username used to log into the system

#### *RS_PASSWORD*

The password used to log into the system

### Optional environment variables

#### *TW_TOKEN*

Your twilio access token

#### *TW_SID*

Your twilio account SID

#### *TW_NUMBER*

Your twilio phone number SID

#### *TO_NUMBER*

The number which will receive delivery SMS messages

---

### Build & Run

`GO111MODULE=on go build github.com/ndane/parcelwatch/cmd/parcelwatch`

`./parcelwatch`

## Todo

- Alexa skill integration
- Local data storage
