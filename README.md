# Cube Datahub

## Getting Started
Expects local MySQL with no password root user.

### DB Migrations
Uses [Skeema](https://www.skeema.io/) for DB migrations. To bootstrap the DB cd to db/cube and run `$ skeema push local`

### Pull a Cube
Get the cube cobra ID for the cube and put it into load.go. If there are any custom cards you need to provide an OPENAI_API_KEY.

### Read a decklist from a picture
Must provide OPENAI_API_KEY then put the URL for an image of the deck in deck.go. Not working particularly well.