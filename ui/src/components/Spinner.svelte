<div class="spinner">
  <div class="bounce1"></div>
  <div class="bounce2"></div>
  <div class="bounce3"></div>
</div>

<style lang="scss">
  .spinner {
    text-align: center;

    > div {
      width: 10px;
      height: 10px;
      background-color: #333;
      // @apply bg-gray-700;

      border-radius: 100%;
      display: inline-block;
      animation: sk-bouncedelay 1.4s infinite ease-in-out both;
    }

    .bounce1 {
      animation-delay: -0.32s;
    }

    .bounce2 {
      animation-delay: -0.16s;
    }
  }

  @screen dark {
    .spinner {
      > div {
        // @apply bg-gray-100;
        background-color: #333;
      }
    }
  }

  @keyframes sk-bouncedelay {
    0%,
    80%,
    100% {
      transform: scale(0);
    }
    40% {
      transform: scale(1);
    }
  }
</style>
