import discord
from discord.ext import commands

import os

bot = commands.Bot(command_prefix=':) ')

@bot.event
async def on_ready():
    await bot.change_presence(activity=discord.Activity(type=discord.ActivityType.watching, name="the Pikmin bloom"))
    print('Alphie is ready to pluck!')

@bot.command()
async def ping(ctx):
    await ctx.send(f'Pong! `{round(bot.latency * 1000)} ms`')

bot.run(os.getenv('API_TOKEN'))
