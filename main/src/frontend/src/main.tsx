import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";

import { App } from "@/app";
import { BRAND_ASSETS } from "@/constants/brand_assets";
import "@/styles/global.css";

const setFavicon = () => {
  const existingLink = document.querySelector<HTMLLinkElement>("link[rel='icon']");
  const faviconLink = existingLink ?? document.createElement("link");

  faviconLink.rel = "icon";
  faviconLink.type = "image/png";
  faviconLink.href = BRAND_ASSETS.icon;

  if (!existingLink) {
    document.head.appendChild(faviconLink);
  }
};

setFavicon();

createRoot(document.getElementById("root") as HTMLElement).render(
  <StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </StrictMode>
);
