import starlightImage from "@resources/img/logo/starlight.png";

import styles from "./system_evolution_hud.module.css";

export const SystemEvolutionHud = () => (
  <div className={styles.hud}>
    <img
      className={styles.hudImage}
      src={starlightImage}
      alt=""
      aria-hidden="true"
    />
  </div>
);
