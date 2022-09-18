<nav>
	<a class="logo" href="/"><img src="/favicon.png" alt="favicon" /></a>
	<a href="/">Home</a>
	<a href="/about">About</a>
</nav>

<slot />

<style>
	:global(body) {
		margin: 0;
	}
	nav {
		height: 60px;
		font-size: 3.5em;
		background-color: #eee;
		padding: 10px;
	}
	nav * {
		height: 100%;
		display: block;
		float: left;
	}
	nav a,
	nav a:active {
		padding-left: 20px;
		font-family: 'Lucida Sans', 'Lucida Sans Regular', 'Lucida Grande', 'Lucida Sans Unicode',
			Geneva, Verdana, sans-serif;
		font-weight: 600;
		color: #666;
		text-decoration: none;
	}
	.logo,
	.logo:active {
		padding: 0;
	}
</style>
