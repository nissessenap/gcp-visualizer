# GCP-visaulizer

A CLI tool built to run from time to time to visualize how GCP resources connect together.
As a second stage, it will also visaulize if the resources is managed through terraform or not.

## main flow

Say that you have a service account, which normally represent an app.
This app is connected to multiple GCP resources, for example

app 1

- pub/sub topic user_creation which it publushes messages to
- A pub/sub subscription called user_event_creator that it listens to topic user_events
- these resources are created in project 1
- A SQL database it writes data to

app 2

- Has a pub/sub subscriptions called user_creation_email that listens to user_creation topic and is created in gcp project 2.

Then it will generate an image where app 1 is connected to the pub/sub topic and a subscription and the database.
App 2 is connected to the topic throug it's subscription.

In short it creates a diagram to show all connections all service accounts/apps got.

## Main issues

- The image can of course become very big after a while
  - I want this resources to span over 1000+ resources
- What visaulization tool should we use
  - Preferably I don't want to build any frontend app to visualize
  - I would prefer using an existing tool that has some kind of API or similar to draw.
    - One option could be to just write markdown, but I don't think that will look good with 10+ resources
- Are there any existing tool that can help out with this?

## MVP

For the MVP I only want to focus on

- GCP SA
- GCP pubsub topics/subscriptions and how they are connected.
