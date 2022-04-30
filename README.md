# Alphie
The best engineer in all of Koppai!

# If you want to contribute to Alphie
Feel free to open a pull request once you have implemented your changes.

You can also open an issue here on GitHub or DM me on Discord if you have feature requests or find a bug.

# If you want to run Alphie yourself
Before running Alphie, a few environment variables have to be set in `bot.env`:
* `API_TOKEN`: The API token for your bot.
* `AUTHORIZED_IDS`: The ID's of the Discord users that should have access to privileged commands. The format should be `ID_0,ID_1,...,ID_n`.
* `HOME_GUILD`: The ID of your main server, this is where the emotes get fetched from.

And in `.env`:
* `POSTGRES_PASSWORD`: The password for accessing the database.

The HOME_GUILD should have an emote named ":alph:".

After that, simply run `docker-compose up` and summon Alphie!