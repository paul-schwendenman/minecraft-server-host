export default {
  content: [
    "./index.html",
    "./src/**/*.{svelte,js,ts}",
    "../../libs/**/*.{svelte,js,ts}",
  ],
  theme: {
    extend: {
      flex: {
        2: "2 2 0%",
      },
    },
  },
  plugins: [],
};
