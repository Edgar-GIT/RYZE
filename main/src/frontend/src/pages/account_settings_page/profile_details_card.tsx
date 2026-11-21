import { FileText, Save } from "lucide-react";
import { type FormEvent, useEffect, useId, useState } from "react";

import { Button } from "@/components/button/button";
import { SOCIAL_NAMES, SOCIAL_PLATFORMS, SocialIcon, type SocialPlatform } from "@/components/social_icons/social_icons";
import { useI18n } from "@/i18n/i18n";
import { fetchProfileDetails, saveProfileDetails } from "@/services/profile_api";
import { joinClassNames } from "@utils/class_names";
import { ApiError } from "@utils/http_client";

import styles from "./profile_details_card.module.css";

const COPY = {
  en: {
    title: "About you",
    lead: "Shown on your public profile when filled in. Everything is optional.",
    bio: "Biography",
    bioHint: "Up to 500 characters.",
    birth: "Date of birth",
    birthHint: "Only your age is shown.",
    since: "Training since (year)",
    weight: "Weight (kg)",
    socials: "Social networks",
    socialsHint: "Tap a network and paste your @username or the link to your profile.",
    socialLabel: (name: string) => `Your ${name} (@username or link)`,
    save: "Save",
    saving: "Saving…",
    saved: "Saved.",
    invalid: "Check the highlighted fields.",
    error: "Something went wrong. Please try again."
  },
  pt: {
    title: "Sobre ti",
    lead: "Aparece no teu perfil público quando preenchido. É tudo opcional.",
    bio: "Biografia",
    bioHint: "Até 500 caracteres.",
    birth: "Data de nascimento",
    birthHint: "Só a tua idade é mostrada.",
    since: "A treinar desde (ano)",
    weight: "Peso (kg)",
    socials: "Redes sociais",
    socialsHint: "Toca numa rede e cola o teu @utilizador ou o link do teu perfil.",
    socialLabel: (name: string) => `O teu ${name} (@utilizador ou link)`,
    save: "Guardar",
    saving: "A guardar…",
    saved: "Guardado.",
    invalid: "Confirma os campos assinalados.",
    error: "Algo correu mal. Tenta de novo."
  }
} as const;

const THIS_YEAR = new Date().getFullYear();

// ProfileDetailsCard edits the public "about" (owner request, 2026-11-21):
// biography, age, years of training, weight and social networks.
export const ProfileDetailsCard = () => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const ids = useId();
  const [loaded, setLoaded] = useState(false);
  const [bio, setBio] = useState("");
  const [birth, setBirth] = useState("");
  const [since, setSince] = useState("");
  const [weight, setWeight] = useState("");
  const [socials, setSocials] = useState<Record<string, string>>({});
  const [open, setOpen] = useState<SocialPlatform | null>(null);
  const [invalid, setInvalid] = useState<string[]>([]);
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    fetchProfileDetails()
      .then((details) => {
        setBio(details.bio);
        setBirth(details.birth_date);
        setSince(details.training_since ? String(details.training_since) : "");
        setWeight(details.weight_kg ? String(details.weight_kg) : "");
        setSocials(Object.fromEntries(details.socials.map((link) => [link.platform, link.handle])));
        setLoaded(true);
      })
      .catch(() => setLoaded(true));
  }, []);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setMessage(null);
    setInvalid([]);
    const weightValue = weight.trim() ? Number(weight.replace(",", ".")) : null;
    try {
      const saved = await saveProfileDetails({
        bio,
        birth_date: birth,
        training_since: since.trim() ? Number(since) : null,
        weight_kg: weightValue,
        socials
      });
      setSocials(Object.fromEntries(saved.socials.map((link) => [link.platform, link.handle])));
      setMessage({ ok: true, text: text.saved });
    } catch (error) {
      if (error instanceof ApiError && error.code === "VALIDATION_ERROR" && error.details.length > 0) {
        setInvalid(error.details);
        const social = error.details.find((field) => field.startsWith("socials."));
        if (social) setOpen(social.slice("socials.".length) as SocialPlatform);
        setMessage({ ok: false, text: text.invalid });
      } else {
        setMessage({ ok: false, text: text.error });
      }
    } finally {
      setBusy(false);
    }
  };

  const bad = (field: string) => invalid.includes(field);

  return (
    <section className={styles.card} aria-labelledby={`${ids}-title`}>
      <h2 id={`${ids}-title`} className={styles.title}>
        <FileText aria-hidden="true" /> {text.title}
      </h2>
      <p className={styles.lead}>{text.lead}</p>

      <form className={styles.form} onSubmit={(event) => void submit(event)} noValidate>
        <label className={joinClassNames(styles.field, bad("bio") && styles.fieldError)}>
          <span>{text.bio}</span>
          <textarea value={bio} maxLength={500} rows={4} disabled={!loaded} onChange={(event) => setBio(event.target.value)} />
          <small>
            {text.bioHint} {bio.length}/500
          </small>
        </label>

        <div className={styles.row}>
          <label className={joinClassNames(styles.field, bad("birth_date") && styles.fieldError)}>
            <span>{text.birth}</span>
            <input type="date" value={birth} disabled={!loaded} onChange={(event) => setBirth(event.target.value)} />
            <small>{text.birthHint}</small>
          </label>
          <label className={joinClassNames(styles.field, bad("training_since") && styles.fieldError)}>
            <span>{text.since}</span>
            <input type="number" inputMode="numeric" min={1950} max={THIS_YEAR} value={since} disabled={!loaded} onChange={(event) => setSince(event.target.value)} />
          </label>
          <label className={joinClassNames(styles.field, bad("weight_kg") && styles.fieldError)}>
            <span>{text.weight}</span>
            <input type="text" inputMode="decimal" value={weight} disabled={!loaded} onChange={(event) => setWeight(event.target.value)} />
          </label>
        </div>

        <fieldset className={styles.socials}>
          <legend>{text.socials}</legend>
          <p className={styles.hint}>{text.socialsHint}</p>
          <div className={styles.icons}>
            {SOCIAL_PLATFORMS.map((platform) => (
              <button
                key={platform}
                type="button"
                className={joinClassNames(
                  styles.icon,
                  socials[platform]?.trim() && styles.iconSet,
                  open === platform && styles.iconOpen,
                  bad(`socials.${platform}`) && styles.iconError
                )}
                aria-label={SOCIAL_NAMES[platform]}
                aria-pressed={open === platform}
                title={SOCIAL_NAMES[platform]}
                onClick={() => setOpen(open === platform ? null : platform)}
              >
                <SocialIcon platform={platform} />
              </button>
            ))}
          </div>
          {open ? (
            <label className={joinClassNames(styles.field, bad(`socials.${open}`) && styles.fieldError)}>
              <span>{text.socialLabel(SOCIAL_NAMES[open])}</span>
              <input
                value={socials[open] ?? ""}
                autoCapitalize="none"
                spellCheck={false}
                placeholder="@"
                onChange={(event) => setSocials({ ...socials, [open]: event.target.value })}
              />
            </label>
          ) : null}
        </fieldset>

        <div className={styles.actions}>
          <Button type="submit" variant="primary" size="small" icon={<Save aria-hidden="true" />} iconPosition="left" disabled={busy || !loaded}>
            {busy ? text.saving : text.save}
          </Button>
          {message ? (
            <span className={message.ok ? styles.ok : styles.error} role={message.ok ? "status" : "alert"}>
              {message.text}
            </span>
          ) : null}
        </div>
      </form>
    </section>
  );
};
