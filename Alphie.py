import discord
from discord.ext import commands

import os
import random
import re

bot = commands.Bot(command_prefix=':) ')

emojis = {
    "alph" : "<:alph:959849626020761680>"
}

@bot.event
async def on_ready():
    await bot.change_presence(activity=discord.Activity(type=discord.ActivityType.watching, name="the Pikmin bloom"))
    print('Alphie is ready to pluck!')

@bot.event
async def on_message(message):
    if message.author.bot:
        return

    # Respond to messages similar to "Hello Alphie!"
    if re.match("^hello alph(ie)?!?$", message.content, re.I):
        responses = ["Hello!", "Who said that?", "Wow, you're huge!", "You're not from Koppai, are you?", "While you're here, can you help me carry this Sunseed Berry?", "Wow, you must be able to throw so many Pikmin at once!"]
        await message.channel.send(random.choice(responses) + " " + str(emojis["alph"]))

    await bot.process_commands(message)

@bot.command()
async def ping(ctx):
    await ctx.send(f'Pong! `{round(bot.latency * 1000)}ms`')

bot.run(os.getenv('API_TOKEN'))
