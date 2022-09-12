# Alphie

The best engineer in all of Koppai!

# If you want to contribute to Alphie

Feel free to open a pull request once you have implemented your changes.

You can also open an issue here on GitHub or DM me on Discord if you have feature requests or find a bug.

# If you want to run Alphie yourself

Before running Alphie, a few environment variables have to be set in `bot.env`:

- `API_TOKEN`: The API token for your bot.
- `AUTHORIZED_IDS`: The ID's of the Discord users that should have access to privileged commands. The format should be `ID_0,ID_1,...,ID_n`.
- `HOME_GUILD`: The ID of your main server, this is where the emotes get fetched from.

And in `.env`:

- `POSTGRES_PASSWORD`: The password for accessing the database.

Also, you need the URL for the lecture streams, which I will give you if you DM me and are a student at ETH.\
The link you'll get must then be added in `grpc.env`:

- `LECTURE_CLIP_BASE_URL`: The base URL for the lecture streams.

The HOME_GUILD should have an emote named ":alph:".

Additionally, the gRPC proto files have to be generated. This can be done by changing to the `rpc` directory and running `make gen`.

After that, simply run `docker-compose --env-file=env/.env up --build` and summon Alphie!

# Working on the Frontend

If you're working on the frontend, instead of having to reload all containers every time you make a change, change into the `www` directory instead and run `yarn dev`. This will cause the `alphie-www` container to hot reload every time you make a change.
Make sure however, that you first run `yarn` or `yarn install` before you start working on the frontend to install all the relevant development dependencies.
