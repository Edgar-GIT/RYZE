import { AtSign, Camera, Lock, Trash2, UserRound } from "lucide-react";
import { Link } from "react-router-dom";
import { type ChangeEvent, type FormEvent, useEffect, useId, useRef, useState } from "react";

import { Button } from "@/components/button/button";
import { type MemberUser } from "@/components/app_shell/member_shell";
import { useI18n } from "@/i18n/i18n";
import { isValidUsername, publicNameOf, removeAvatar, updateProfile, uploadAvatar } from "@/services/profile_api";
import { profilePath } from "@/services/public_profiles_api";
import { ApiError, publicFileUrl } from "@utils/http_client";

import styles from "./public_profile_card.module.css";

const COPY = {
  en: {
    title: "Public profile",
    view: "View my public profile",
    lead: "How other people see you on RYZE: your nickname, or your @username when you have none.",
    photo: "Profile photo",
    change: "Change photo",
    remove: "Remove photo",
    photoHint: "JPEG or PNG. It is cropped to a square and its hidden details (such as location) are removed.",
    displayName: "Nickname",
    displayHint: "Up to 40 characters. Leave it empty to show your @username. You can change it once every 7 days.",
    nicknameLocked: (date: string) => `You changed your nickname recently. You can change it again on ${date}.`,
    username: "@username",
    usernameHint: "3 to 30 lowercase letters, digits, dots or underscores.",
    usernameFixed: "Your @username is unique to your account and never changes, like your email.",
    save: "Save profile",
    saved: "Profile saved.",
    photoSaved: "Photo updated.",
    errors: {
      USERNAME_TAKEN: "This @username is already in use.",
      USERNAME_LOCKED: "Your @username cannot be changed.",
      NICKNAME_COOLDOWN: "You can change your nickname once every 7 days.",
      INVALID_USERNAME: "An @username has 3 to 30 lowercase letters, digits, dots or underscores.",
      INVALID_DISPLAY_NAME: "A nickname has 1 to 40 characters.",
      INVALID_IMAGE: "The photo must be a JPEG or PNG image.",
      IMAGE_TOO_LARGE: "The photo is too large.",
      generic: "Something went wrong. Please try again."
    } as Record<string, string>
  },
  pt: {
    title: "Perfil público",
    view: "Ver o meu perfil público",
    lead: "Como os outros te veem na RYZE: a tua alcunha ou, se não tiveres, a tua @referência.",
    photo: "Foto de perfil",
    change: "Mudar foto",
    remove: "Remover foto",
    photoHint: "JPEG ou PNG. É recortada em quadrado e os dados escondidos (como a localização) são removidos.",
    displayName: "Alcunha",
    displayHint: "Até 40 caracteres. Deixa vazio para mostrar a tua @referência. Podes mudá-la uma vez a cada 7 dias.",
    nicknameLocked: (date: string) => `Mudaste a alcunha há pouco tempo. Podes voltar a mudá-la a ${date}.`,
    username: "@referência",
    usernameHint: "3 a 30 letras minúsculas, números, pontos ou underscores.",
    usernameFixed: "A tua @referência é única da tua conta e nunca muda, tal como o email.",
    save: "Guardar perfil",
    saved: "Perfil guardado.",
    photoSaved: "Foto atualizada.",
    errors: {
      USERNAME_TAKEN: "Esta @referência já está a ser usada.",
      USERNAME_LOCKED: "A tua @referência não pode ser alterada.",
      NICKNAME_COOLDOWN: "Só podes mudar a alcunha uma vez a cada 7 dias.",
      INVALID_USERNAME: "A @referência tem 3 a 30 letras minúsculas, números, pontos ou underscores.",
      INVALID_DISPLAY_NAME: "A alcunha tem de 1 a 40 caracteres.",
      INVALID_IMAGE: "A foto tem de ser uma imagem JPEG ou PNG.",
      IMAGE_TOO_LARGE: "A foto é demasiado grande.",
      generic: "Algo correu mal. Tenta de novo."
    } as Record<string, string>
  }
} as const;

export const initialsFrom = (name: string) =>
  name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("") || "R";

// ProfileAvatar draws the photo, or the initials of the public name.
export const ProfileAvatar = ({ user, className }: { user: MemberUser; className?: string }) =>
  user.avatar_url ? (
    <img className={className} src={publicFileUrl(user.avatar_url)} alt="" />
  ) : (
    <span className={className} aria-hidden="true">
      {initialsFrom(publicNameOf(user))}
    </span>
  );

// PublicProfileCard edits the identity people see (owner request,
// 2026-11-20): the photo, the display name and the @username.
export const PublicProfileCard = ({ user, onChanged }: { user: MemberUser; onChanged: () => void }) => {
  const { locale, formatDate } = useI18n();
  const text = COPY[locale];
  const ids = useId();
  // The @username is permanent once chosen (owner decision, 2026-11-23); the
  // nickname changes at most once every 7 days (enforced by the server).
  const usernameFixed = !user.needs_username;
  const nicknameLockedUntil =
    user.nickname_change_at && Date.parse(user.nickname_change_at) > Date.now() ? user.nickname_change_at : null;
  const fileRef = useRef<HTMLInputElement>(null);
  const [displayName, setDisplayName] = useState(user.display_name ?? "");
  const [username, setUsername] = useState(user.username);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ kind: "ok" | "error"; text: string } | null>(null);

  useEffect(() => {
    setDisplayName(user.display_name ?? "");
    setUsername(user.username);
  }, [user.display_name, user.username]);

  const errorText = (error: unknown) => (error instanceof ApiError && error.code ? text.errors[error.code] : undefined) ?? text.errors.generic;
  const usernameValid = usernameFixed || isValidUsername(username);

  const run = async (action: () => Promise<unknown>, success: string) => {
    setBusy(true);
    setMessage(null);
    try {
      await action();
      setMessage({ kind: "ok", text: success });
      onChanged();
    } catch (error) {
      setMessage({ kind: "error", text: errorText(error) });
    } finally {
      setBusy(false);
    }
  };

  const save = (event: FormEvent) => {
    event.preventDefault();
    if (!usernameValid) {
      setMessage({ kind: "error", text: text.errors.INVALID_USERNAME });
      return;
    }
    void run(() => updateProfile(displayName.trim(), usernameFixed ? user.username : username.trim().replace(/^@/, "").toLowerCase()), text.saved);
  };

  const pick = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (file) void run(() => uploadAvatar(file), text.photoSaved);
  };

  return (
    <section className={styles.card} aria-labelledby={`${ids}-title`}>
      <h2 id={`${ids}-title`} className={styles.title}>
        <UserRound aria-hidden="true" /> {text.title}
      </h2>
      <p className={styles.lead}>
        {text.lead}{" "}
        <Link className={styles.profileLink} to={profilePath(user.username)}>
          {text.view}
        </Link>
      </p>

      <div className={styles.photoRow}>
        <ProfileAvatar user={user} className={styles.avatar} />
        <div className={styles.photoActions}>
          <span className={styles.label}>{text.photo}</span>
          <div className={styles.buttons}>
            <Button type="button" variant="secondary" size="small" icon={<Camera aria-hidden="true" />} iconPosition="left" disabled={busy} onClick={() => fileRef.current?.click()}>
              {text.change}
            </Button>
            {user.avatar_url ? (
              <Button type="button" variant="ghost" size="small" icon={<Trash2 aria-hidden="true" />} iconPosition="left" disabled={busy} onClick={() => void run(removeAvatar, text.photoSaved)}>
                {text.remove}
              </Button>
            ) : null}
          </div>
          <p className={styles.hint}>{text.photoHint}</p>
          <input ref={fileRef} className={styles.file} type="file" accept="image/jpeg,image/png" onChange={pick} tabIndex={-1} aria-hidden="true" />
        </div>
      </div>

      <form className={styles.form} onSubmit={save} noValidate>
        <label className={styles.field}>
          <span className={styles.label}>{text.displayName}</span>
          <input
            value={displayName}
            maxLength={40}
            autoComplete="nickname"
            placeholder={`${user.first_name} ${user.last_name}`.trim()}
            onChange={(event) => setDisplayName(event.target.value)}
            readOnly={Boolean(nicknameLockedUntil)}
            aria-describedby={`${ids}-display-hint`}
          />
          <span id={`${ids}-display-hint`} className={styles.hint}>
            {nicknameLockedUntil
              ? text.nicknameLocked(formatDate(nicknameLockedUntil, { day: "numeric", month: "long", hour: "2-digit", minute: "2-digit" }))
              : text.displayHint}
          </span>
        </label>
        <label className={styles.field}>
          <span className={styles.label}>{text.username}</span>
          <span className={styles.prefixed}>
            <AtSign aria-hidden="true" />
            <input
              value={username}
              maxLength={31}
              autoComplete="username"
              autoCapitalize="none"
              spellCheck={false}
              onChange={(event) => setUsername(event.target.value.toLowerCase())}
              readOnly={usernameFixed}
              aria-invalid={!usernameValid}
              aria-describedby={`${ids}-username-hint`}
            />
            {usernameFixed ? <Lock className={styles.lock} aria-hidden="true" /> : null}
          </span>
          <span id={`${ids}-username-hint`} className={usernameValid ? styles.hint : styles.error}>
            {usernameFixed ? text.usernameFixed : usernameValid ? text.usernameHint : text.errors.INVALID_USERNAME}
          </span>
        </label>
        {message ? (
          <p className={message.kind === "ok" ? styles.ok : styles.error} role={message.kind === "error" ? "alert" : "status"}>
            {message.text}
          </p>
        ) : null}
        <div>
          <Button type="submit" disabled={busy || !usernameValid}>
            {text.save}
          </Button>
        </div>
      </form>
    </section>
  );
};
