
<script>
    import ServerDetails from './ServerDetails.svelte';
	import { getDetails } from '../data.service';

    export let serverStatus;
    export let handleStart;
    export let handleStop;
    export let handleRefresh;
    export let handleSyncDNS;
</script>

<header>
    <h1>{serverStatus.dns_record.name}</h1>
</header>
<p>
    Server is {serverStatus.instance.state}.
</p>
{#if serverStatus.instance.state == "stopped"}
    <button on:click={handleStart} class="button">
        Start
    </button>
{:else if serverStatus.instance.state == "running"}
    <p>IP address: <code>{serverStatus.instance.ip_address}</code></p>

    {#if serverStatus.instance.ip_address != serverStatus.dns_record.value}
        <button on:click={handleSyncDNS} class="button">
            Update DNS Record
        </button>
    {:else}
        <ServerDetails serverDetails={getDetails(serverStatus.instance.ip_address)} />
    {/if}
    <button on:click={handleStop} class="button">
        Stop
    </button>
{/if}
<button on:click={handleRefresh} class="button">
    Refresh
</button>
