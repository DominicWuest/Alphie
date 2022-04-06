# Alphie
My little baby :3

# If you want to run Alphie yourself
Enter the Alphie directory and run
```
pip install -r requirements.txt
```

Before running the main file Alphie.py itself, a few environment variables are required:
* API_TOKEN: The API token for your bot.
* AUTHORIZED_IDS: The ID's of the Discord users that should have access to privileged commands. The format should be `ID_0,ID_1,...,ID_n`.
* HOME_GUILD: The ID of your main server, this is where the emotes get fetched from.

The HOME_GUILD should have an emote named "alph".