import discord
from discord.ext import commands

class Blackjack(commands.Cog):

    def __init__(self, bot):
        self.bot = bot
        self.player = None

    @commands.command()
    async def blackjack(self, ctx):
        if (self.player == None):
            await ctx.send("Started Blackjack")

def setup(bot):
    bot.add_cog(Blackjack(bot))