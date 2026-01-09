<script>
  import '../app.css';
</script>

<div class="app-container">
  <slot />
</div>

<style>
  .app-container {
    height: 100vh;
    width: 100vw;
    display: flex;
    flex-direction: column;
    background-color: var(--bg-color);
  }
</style>
