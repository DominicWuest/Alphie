import os

needed_vars = ['COMMON_DOMAIN', 'STUDENT_AUTH_PATH',
               'CDN_DOMAIN', 'DEV_MAIL_ADDR']


def error(var):
    raise Exception(var + ' is not defined, aborted')


list(map(lambda x: error(x) if not os.getenv(x) else None, needed_vars))

if os.getenv('STUDENT_AUTH_ENABLED'):
    # Ensure JWT public key and authorization url is set if auth is enabled
    if not (os.getenv('JWT_PUBLIC_KEY') and os.getenv('AUTHORIZATION_URL')):
        error('JWT_PUBLIC_KEY')

print('Environment variable checking succeeded. Proceeding...')
