<script lang="ts">
	import type { PageData } from './$types';
	import VideoPlayer from 'svelte-video-player';

	export let data: PageData;
</script>

<div class="video-wrapper">
	<VideoPlayer poster="/favicon.png" source="http://{data.domain}/lecture_clips/{data.id}.mp4" />
</div>

<footer>
	The clips are from livestreams broadcasted at ETH and are thus licensed under <a
		href="https://creativecommons.org/licenses/by-nc-nd/2.5/ch/deed.en_US">CC BY-NC-ND 2.5 CH</a
	>. <br />
	All video content is password protected and only accessibly by students with a valid login. <br />
	For any inquiries, please contact the developer via mail
	<a href="mailto:{data.mail}">here</a>.
</footer>

<style>
	.video-wrapper {
		width: 50%;
		margin: auto;
		margin-top: 50px;
	}
	footer {
		text-align: center;
		font-family: 'Lucida Sans', 'Lucida Sans Regular', 'Lucida Grande', 'Lucida Sans Unicode',
			Geneva, Verdana, sans-serif;
		padding-bottom: 5px;
		height: auto;
		width: 100%;
		position: absolute;
		bottom: 0;
	}
</style>
