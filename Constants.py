import os

emojis = {
    "alph" : "<:alph:959849626020761680>"
}

authorized_ids = []
# AUTHORIZED_IDS should be in the form of ID_0,ID_1,...,ID_n
for id in os.getenv('AUTHORIZED_IDS').split(','):
    authorized_ids.append(int(id))