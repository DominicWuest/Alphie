import os

needed_vars = ['STUDENT_AUTH_PATH', 'CDN_DOMAIN', 'DEV_MAIL_ADDR']


def error(var):
    raise Exception(var + ' is not defined, aborted')


list(map(lambda x: error(x) if not os.getenv(x) else None, needed_vars))

if os.getenv('STUDENT_AUTH_ENABLED'):
    # Ensure JWT public key is set if auth is enabled
    if not os.getenv('JWT_PUBLIC_KEY'):
        error('JWT_PUBLIC_KEY')

print('Environment variable checking succeeded. Proceeding...')
