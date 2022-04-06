import discord
from discord.ext import commands

import time
import random

from Constants import emojis

CARDS = ["2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"]

# Game state constants
IDLE = 0
DEALING = 1
WAITING = 2
OVER = 3

EMBED_COLOUR = discord.Colour.dark_gold()

class Blackjack(commands.Cog):

    def __init__(self, bot):
        self.bot = bot
        # The game state
        self.state = IDLE
        # The user playing the game
        self.player = None
        # The message in which the game gets displayed
        self.game_message = None
        # The possible totals that can be achieved with the players cards
        self.player_totals = []
        # The players hand
        self.player_cards = []
        # The possible totals that can be achieved with the dealers cards
        self.dealer_totals = []
        # The dealers hand
        self.dealer_cards = []
        # The currently still available cards: Four sets of normal cards minus the ones that have been dealt
        self.curr_cards = []

    # Generates an embed depending on what the state of the game is
    def gen_embed(self, state):
        '''
            state < 0 => player lost
            state = 0 => game ongoing
            state > 0 => player won
        '''
        embed = discord.Embed(
            colour=EMBED_COLOUR
        ).set_author(
            name="Blackjack: " + ("Dealing..." if self.state == DEALING else "")
        ).set_thumbnail(
            url="https://media.istockphoto.com/photos/blackjack-spades-picture-id155428832"
        ).add_field(
            name="Dealer's Hand",
            value=" ".join(self.dealer_cards) if len(self.dealer_cards) != 0 else "Empty",
            inline=False
        ).add_field(
            name="Your Hand",
            value=" ".join(self.player_cards) if len(self.player_cards) != 0 else "Empty",
            inline=True
        ).set_footer(
            text="Invoked by " + self.player.display_name,
            icon_url=self.player.avatar_url
        )

        if state == 0:
            embed.add_field(
                name="Controls",
                value=f'{emojis["play"]} To Hit\n{emojis["pause"]} To Stand\n{emojis["fail"]} To Cancel Game',
                inline=False
            )
        else:
            embed.add_field(
                name="You " + ("Won!" if state > 0 else "Lost..."),
                value=f'{emojis["repeat"]} To Play Again\n{emojis["fail"]} To Stop Playing',
                inline=False
            )

        if state < 0:
            embed.set_image(url="http://cdn140.picsart.com/264364272004202.png")
        elif state > 0:
            embed.set_image(url="https://i.redd.it/bl3s4acqqgq31.png")

        return embed

    # Starts a new game of blackjack, ctx should be set if a new game is started
    async def start_blackjack(self, player, ctx=None):
        self.state = DEALING

        self.player = player

        self.player_totals = [0]
        self.player_cards = []
        self.dealer_totals = [0]
        self.dealer_cards = []

        self.curr_cards = 4 * CARDS[:] # Dereference array

        # New game
        if self.game_message == None:
            self.game_message = await ctx.send(embed=self.gen_embed(0))
        else:
            await self.game_message.edit(embed=self.gen_embed(0))

        # Add the reactions for playing
        await self.game_message.add_reaction(emojis["play"])
        await self.game_message.add_reaction(emojis["pause"])
        await self.game_message.add_reaction(emojis["fail"])

        # Initial deal
        self.deal(True)
        await self.game_message.edit(embed=self.gen_embed(0))

        time.sleep(0.5)
        self.deal(True)
        await self.game_message.edit(embed=self.gen_embed(0))
        
        time.sleep(0.5)
        self.deal(False)
        await self.game_message.edit(embed=self.gen_embed(0))

        # If the player got blackjack
        if 21 in self.player_totals:
            await self.game_message.edit(embed=self.gen_embed(1))

            await self.game_message.clear_reactions()
            await self.game_message.add_reaction(emojis["repeat"])
            await self.game_message.add_reaction(emojis["fail"])

            self.state = OVER

            return

        self.state = WAITING
        await self.game_message.edit(embed=self.gen_embed(0))

    # Deals one card to either the player (player = True) or the dealer (player = False)
    def deal(self, player):
        index = random.randint(0, len(self.curr_cards) - 1)
        val = self.curr_cards[index]
        self.curr_cards.pop(index)

        cards = self.player_cards if player else self.dealer_cards
        totals = self.player_totals if player else self.dealer_totals

        cards.append(val)

        if val != "A":
            intval = 10 if val in ("J", "Q", "K") else int(val)
            totals = [i + intval for i in totals]
        else:
            totals = [i + 11 for i in totals] + [i + 1 for i in totals]
        
        totals = [i for i in totals if i <= 21]

        if player:
            self.player_totals = totals
        else:
            self.dealer_totals = totals

    # Stops the game and resets
    async def cancel_game(self):
        embed = discord.Embed(
            colour = EMBED_COLOUR
        ).set_author(
            name="Blackjack"
        ).add_field(
            name="Game Stopped",
            value=self.player.mention + " has stopped the game. Thanks for playing!"
        ).set_footer(
            text="Invoked by " + self.player.display_name,
            icon_url=self.player.avatar_url
        )

        await self.game_message.edit(embed=embed)
        await self.game_message.clear_reactions()

        self.state = IDLE

    @commands.Cog.listener()
    async def on_reaction_add(self, reaction, user):
        # If the reaction got added to a different message or someone other than the player added it
        if reaction.message != self.game_message or user != self.player:
            return

        # If user stops game
        if str(reaction.emoji) == emojis["fail"]:
            await self.cancel_game()
            return
        
        # If user restarts game
        if str(reaction.emoji) == emojis["repeat"] and self.state == OVER:
            await self.game_message.clear_reactions()
            await self.start_blackjack(self.player)

        # If the reaction is not the right emote or the bot isn't listening for emotes currently (because it's dealing)
        if self.state != WAITING or str(reaction.emoji) not in (emojis["pause"], emojis["play"]):
            await reaction.remove(user)
            return

        # Hit
        if str(reaction.emoji) == emojis["play"]:
            self.state = DEALING
            await self.game_message.edit(embed=self.gen_embed(0))

            self.deal(True)
            await self.game_message.edit(embed=self.gen_embed(0))

            self.state = WAITING

            # Player went bust
            if len(self.player_totals) == 0:
                await self.game_message.edit(embed=self.gen_embed(-1))

                await self.game_message.clear_reactions()
                await self.game_message.add_reaction(emojis["repeat"])
                await self.game_message.add_reaction(emojis["fail"])

                self.state = OVER

                return
            # Player got 21
            elif 21 in self.player_totals:
                await self.game_message.edit(embed=self.gen_embed(1))

                await self.game_message.clear_reactions()
                await self.game_message.add_reaction(emojis["repeat"])
                await self.game_message.add_reaction(emojis["fail"])

                self.state = OVER

                return
            
            await self.game_message.edit(embed=self.gen_embed(0))

            await reaction.remove(user)

        # Stand
        elif str(reaction.emoji) == emojis["pause"]:
            # As long as the dealer has either a lower total score than the player and the dealer didn't go bust
            while len(self.dealer_totals) != 0 and max(self.dealer_totals) < max(self.player_totals):
                time.sleep(0.5)
                self.deal(False)
                await self.game_message.edit(embed=self.gen_embed(0))

            # Dealer went bust
            if len(self.dealer_totals) == 0:
                await self.game_message.edit(embed=self.gen_embed(1))

                await self.game_message.clear_reactions()
                await self.game_message.add_reaction(emojis["repeat"])
                await self.game_message.add_reaction(emojis["fail"])

                self.state = OVER

                return
            # Player lost
            else:
                await self.game_message.edit(embed=self.gen_embed(-1))

                await self.game_message.clear_reactions()
                await self.game_message.add_reaction(emojis["repeat"])
                await self.game_message.add_reaction(emojis["fail"])

                self.state = OVER

                return

    @commands.command()
    async def blackjack(self, ctx):
        if (self.state == IDLE):
            await ctx.message.add_reaction(emojis["success"])
            await ctx.message.delete(delay=2)

            await self.start_blackjack(ctx.author, ctx)
        else:
            await ctx.message.add_reaction(emojis["fail"])
            await ctx.message.reply("Sorry, someone else is playing already", delete_after=2)
            await ctx.message.delete(delay=2)

def setup(bot):
    bot.add_cog(Blackjack(bot))