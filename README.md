# Alphie

The best engineer in all of Koppai!

# If you want to contribute to Alphie

Feel free to open a pull request once you have implemented your changes.

You can also open an issue here on GitHub or DM me on Discord if you have feature requests or find a bug.

# If you want to run Alphie yourself

Before running Alphie, a few environment variables have to be set:

 <table>
  <tr>
    <th>File</th>
    <th>Variable</th>
    <th>Description</th>
  </tr>
  <tr>
    <td rowspan="3"><code>bot.s.env</code></td>
    <td><code>API_TOKEN</code></td>
    <td>The API token for your bot.</td>
  </tr>
  <tr>
    <td><code>AUTHORIZED_IDS</code></td>
    <td>The IDs of the Discord users that should have access to privileged commands. The format should be <code>ID_0, ID_1, ..., ID_n</code>.</td>
  </tr>
  <tr>
    <td><code>HOME_GUILD</code></td>
    <td>The ID of your main server, this is where the emotes get fetched from.</td>
  </tr>
  <tr>
    <td rowspan="2"><code>db.s.env</code></td>
    <td><code>POSTGRES_USER</code></td>
    <td>The username for accessing the database.</td>
  </tr>
  <tr>
    <td><code>POSTGRES_PASSWORD</code></td>
    <td>The password for accessing the database.</td>
  </tr>
  <tr>
    <td><code>grpc.s.env</code></td>
    <td><code>LECTURE_CLIP_BASE_URL</code></td>
    <td>The base URL to the lecture streams. I will give you this link if you DM me and are a student at ETH.</td>
  </tr>
  <tr>
    <td><code>www.env</code></td>
    <td><code>STUDENT_AUTH_ENABLED</code></td>
    <td>Whether student authentication should be enabled when accessing lecture clips. This has to be set to a non-null value in production.</td>
  </tr>
  <tr>
    <td rowspan="2"><code>www.s.env</code></td>
    <td><code>DEV_MAIL_ADDR</code></td>
    <td>An E-Mail address over which you are reachable.</td>
  </tr>
  <tr>
    <td><code>AUTHORIZATION_URL</code></td>
    <td>This URL has to point to where you host the files inside the `auth` directory. You can read more about this in the <a href="#securing-the-lecture-clips">Security</a> section.</td>
  </tr>
</table>

Additionally, the gRPC proto files have to be generated. This can be done by changing to the `rpc` directory and running `make gen`.

When running Alphie locally, make sure the domains (`COMMON_DOMAIN`, `CDN_DOMAIN` & `WWW_DOMAIN`) point to localhost.

The variables inside `s.env` files are secrets and should thus not be pushed to your repository. Ensure this by executing `git update-index --assume-unchanged env/*.s.env`.

Now you are ready to run `docker-compose --env-file=env/.env up --build` and summon Alphie!

# Working on the Frontend

If you're working on the frontend, instead of having to reload all containers every time you make a change, change into the `www` directory instead and run `yarn dev`. This will cause the `alphie-www` container to hot reload every time you make a change.
Make sure however, that you first run `yarn` or `yarn install` before you start working on the frontend to install all the relevant development dependencies.

# Securing the Lecture Clips

Make sure you followed the steps in the [If you want to run Alphie yourself](#if-you-want-to-run-alphie-yourself) section first, before proceeding with the next steps.

The RSA keys inside `auth/key.txt` and `env/www.env` are example keys for the purpose of testing and documented publicly. Do not use them in production under any circumstances.

In order for the lecture clip authentication to work, you have to generate a new RSA256 keypair and store the private key in `auth/key.txt` and the public key in `env/www.env`.

Next, host the files contained in the `auth` directory and point the `AUTHORIZATION_URL` to it. You might need to adjust the `.htaccess.n` file first in order for authorization to work. If you are a student of ETH, you can simply host them on your personal website.

Make sure your application runs over https in production by specifying the protocol inside `env/.env` and creating certificates for the domains.
