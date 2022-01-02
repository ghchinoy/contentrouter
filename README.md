# contentrouter

A Cloud Run service to gate static content stored in Google Cloud Storage access via Firebase Auth and Firebase Hosting.

## Background

Firestore Hosting is a fantastic web content hosting though it doesn't have direct integration with Firebase Authentication to restrict access to only authenticated users.

In order to restrict access to protected static content, configure Firebase Hosting rewrite rules to route to this content router Cloud Run service to gate access to static content.

Firebase Hosting will be used to serve the sample login/logout pages (and any other public, unauthenticated access pages) as well as rewrite / redirect URLs to be protected to the Cloud Run contentrouter service.

Google Cloud Storage will contain the protected, restricted static content and will only be served via the content routerservice if the Firebase Authentication token provided is valid.

## Steps

* Set up Firebase Hosting
* Deploy the sample files to Firebase Hosting
* modify the firebase.json to add the rewrite rule below that routes to the Cloud Run service named `contentrouter`
* copy the protected content to a Google Cloud Storage bucket subdirectory
* Deploy the Cloud Run contentrouter service configured with the bucket name and route map, with an appropriate service account


## Detailed steps

### create a service account with roles

Via Firebase's console, one can create a Firebase Admin SDK service account. In the Google Cloud Console, you'll have to add the Storage Reader role.


### Rewrite rule for Cloud Run service

if the Cloud Run service is 1) named `contentrouter`, 2_ is deployed in the region us-central1 and 3) you'd like to authenticate content under the URI `/content`, this is what your Firebase Hosting firebase.json should have added to it, for the `rewrites` rule.

```
    {
      "public": "sample",
      "rewrites": [
        {
          "source": "/content{,/**}",
          "run": {
            "serviceId": "contentrouter",  
            "region": "us-central1" 
          }
        }
      ]
    }
```


### Deploy contentrouter Cloud Run service

```
export SERVICE_ACCOUNT=
export PROJECT_ID=
export REGION=us-central1

export SERVICE_NAME=contentrouter
export BUCKET=secret-bucket
export FIREBASEPATH="content/"
export GCSPATH="restricted/"
export REDIRECTPATH="/index.html"

gcloud run deploy ${SERVICE_NAME} --source . \ 
 --service-account ${SERVICE_ACCOUNT}@${PROJECT_ID}.iam.gserviceaccount.com \
 --set-env-vars "BUCKET=${BUCKET}" \
 --set-env-vars "FIREBASEPATH=${FIREBASEPATH}" \
 --set-env-vars "GCSPATH=${GCSPATH}" \
 --set-env-vars "REDIRECTPATH=${REDIRECTPATH}" \
 --region ${REGION} \
 --allow-unauthenticated
```

* SERVICE_NAME - required, Cloud Run service name, also referenced in firebase.json rewrite run rule
* BUCKET - required, base GCS path, do not include "gs://"
* FIREBASEPATH - the path in firebase.json that redirects to contentrouter
* GCSPATH - the subdir within BUCKET where content is located
* REDIRECTPATH - when user is unauthenticated, redirect back to this path, defaults to "/"

## What contentrouter does

The contentrouter service does three things:

1. Validates the authentication token from Firebase Authentication 
2. Sets the token as a session cookie
3. Serves content from Google Cloud Storage bucket, with an optional simple rewrite of its own



