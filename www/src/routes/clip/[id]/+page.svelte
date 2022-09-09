<script lang="ts">
	import type { PageData } from './$types';
	import VideoPlayer from 'svelte-video-player';

	export let data: PageData;
</script>

<VideoPlayer source="http://{data.domain}/lecture_clips/{data.id}.mp4" />
