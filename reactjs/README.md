# reactjs

An xxdk example written for React.

## Running This Example

You must have a modern `node` and `npm` installation:

https://docs.npmjs.com/downloading-and-installing-node-js-and-npm

Run the npm install command to install dependencies:

```
npm i
```

Now install a symbolic link to `node_modules/xxdk-wasm` inside your public folder. This will serve the `wasm` files needed for `xxdk` to run in the browser from your local machine:

```
cd public
ln -s ../node_modules/xxdk-wasm xxdk-wasm
cd ..
```

Start the example:

```
npm run dev
```

You can now visit http://localhost:3000 on your local browser to view
the example app.

## How This Example Was Built

We used the basic `create-nextjs-app` tool:

* https://nextjs.org/learn-pages-router/basics/create-nextjs-app

Then we added Next UI:

* https://nextui.org/docs/guide/introduction

We then did a basic design, adding XXDK, console-feed, and the Dexie
library afterward and subsequently building out the components:

```
npm i xxdk-wasm --save
npm i dexie --save
npm i console-feed --save
```

Note that `console-feed` is not used in the final example but you can
see the `XXLogs` component. It was used to view the console logs
instead of manually opening the console.

## NextJS Instructions

This is a [Next.js](https://nextjs.org/) project bootstrapped with [`create-next-app`](https://github.com/vercel/next.js/tree/canary/packages/create-next-app).

### Getting Started

First, run the development server:

```bash
npm run dev
# or
yarn dev
# or
pnpm dev
# or
bun dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

You can start editing the page by modifying `app/page.tsx`. The page auto-updates as you edit the file.

This project uses [`next/font`](https://nextjs.org/docs/basic-features/font-optimization) to automatically optimize and load Inter, a custom Google Font.

### Learn More

To learn more about Next.js, take a look at the following resources:

- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.

You can check out [the Next.js GitHub repository](https://github.com/vercel/next.js/) - your feedback and contributions are welcome!

### Deploy on Vercel

The easiest way to deploy your Next.js app is to use the [Vercel Platform](https://vercel.com/new?utm_medium=default-template&filter=next.js&utm_source=create-next-app&utm_campaign=create-next-app-readme) from the creators of Next.js.

Check out our [Next.js deployment documentation](https://nextjs.org/docs/deployment) for more details.
