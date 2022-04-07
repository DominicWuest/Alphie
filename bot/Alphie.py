import discord
from discord.ext import commands

import Constants
from Constants import emojis
from Constants import authorized_ids

import os
import random
import re

bot = commands.Bot(command_prefix=':) ')

@bot.event
async def on_ready():
    await Constants.initialize_constants(bot)
    await bot.change_presence(activity=discord.Activity(type=discord.ActivityType.watching, name="the Pikmin bloom"))
    print('Alphie is ready to pluck!')

@bot.event
async def on_message(message):
    if message.author.bot:
        return

    # Respond to messages similar to "Hello Alphie!"
    if re.match("^hello alph(ie)?!?$", message.content, re.I):
        responses = ["Hello!", "Who said that?", "Wow, you're huge!", "You're not from Koppai, are you?", "While you're here, can you help me carry this Sunseed Berry?", "Wow, you must be able to throw so many Pikmin at once!"]
        await message.channel.send(random.choice(responses) + " " + emojis["alph"])

    await bot.process_commands(message)

@bot.command()
async def ping(ctx):
    await ctx.send(f'Pong! `{round(bot.latency * 1000)}ms` {emojis["alph"]}')

@bot.command()
# Load extension
async def load(ctx, module):
    if (ctx.author.id in authorized_ids):
        try:
            bot.load_extension('cogs.' + module)
            await ctx.send(f'Successfully loaded {module} {emojis["alph"]}')
        except:
            await ctx.send(f'Couldn\'t load {module} {emojis["alph"]}')
    else:
        await ctx.send('You haven\'t even thrown a Pikmin yet and already try to use such a command? Permission denied. ' + emojis["alph"])
    
@bot.command()
# Unload extension
async def unload(ctx, module):
    if (ctx.author.id in authorized_ids):
        try:
            bot.unload_extension('cogs.' + module)
            await ctx.send(f'Successfully unloaded {module} {emojis["alph"]}')
        except:
            await ctx.send(f'Couldn\'t unload {module} {emojis["alph"]}')
    else:
        await ctx.send('You haven\'t even thrown a Pikmin yet and already try to use such a command? Permission denied. ' + emojis["alph"])

# Load all cogs on startup
for cog in os.listdir('cogs'):
    if (cog.endswith(".py")):
        bot.load_extension('cogs.' + cog[:-3])

bot.run(os.getenv('API_TOKEN'))