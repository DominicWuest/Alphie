import os

# The ID of the home guild 
__HOME_GUILD_ID = 0
# The actual object of the home guild
__HOME_GUILD = None

# A dictionary of emojis from the home guild
emojis = {
    "success" : "✅",
    "fail" : "❌",
    "pause" : "⏸️",
    "play" : "▶️",
    "repeat" : "🔁"
}

# The ID's of users who are allowed to use privileged commands
authorized_ids = []

# Actually initializes the constants
async def initialize_constants(bot):
    # Read the ID of the home guild and get it's object
    __HOME_GUILD_ID = int(os.getenv('HOME_GUILD'))
    __HOME_GUILD = await bot.fetch_guild(__HOME_GUILD_ID)

    # Add all emojis of the home guild
    for emoji in __HOME_GUILD.emojis:
        emojis[emoji.name] = str(emoji)

    # AUTHORIZED_IDS should be in the form of ID_0,ID_1,...,ID_n
    for id in os.getenv('AUTHORIZED_IDS').split(','):
        authorized_ids.append(int(id))