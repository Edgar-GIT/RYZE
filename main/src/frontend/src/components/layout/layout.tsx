import type { ReactNode } from "react";

import { AnimatedBackground } from "@/components/animated_background/animated_background";
import { Footer } from "@/components/footer/footer";
import { Navbar } from "@/components/navbar/navbar";

import styles from "./layout.module.css";

interface LayoutProps {
  children: ReactNode;
}

export const Layout = ({ children }: LayoutProps) => (
  <>
    <a className={styles.skipLink} href="#main-content">
      Skip to content
    </a>
    <AnimatedBackground />
    <Navbar />
    <main id="main-content" className={styles.main}>
      {children}
    </main>
    <Footer />
  </>
);
