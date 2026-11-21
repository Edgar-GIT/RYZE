import { AlertTriangle, FileUp, Plus, Send, ShieldCheck, Trash2, X } from "lucide-react";
import { type ChangeEvent, type FormEvent, type ReactElement, useId, useState } from "react";

import { Button } from "@/components/button/button";
import { useI18n } from "@/i18n/i18n";
import {
  submitTrainerApplication,
  type TrainerApplicationFiles,
  type TrainerDocumentRejection,
  type TrainerEducationEntry,
  type TrainerWorkEntry
} from "@/services/trainer_api";
import { ApiError } from "@utils/http_client";
import { joinClassNames } from "@utils/class_names";

import styles from "./trainer_application_form.module.css";

type Kind = "certification" | "identification" | "insurance" | "other";
const MAX_FILES = 6;
const MAX_FILE_BYTES = 10 * 1024 * 1024;
const ACCEPT = ".pdf,.jpg,.jpeg,.png,application/pdf,image/jpeg,image/png";
const THIS_YEAR = new Date().getFullYear();

const COPY = {
  en: {
    title: "Apply to become a RYZE trainer",
    lead: "Tell us who you are and show your qualifications. Every application is checked automatically and reviewed by a person before any decision.",
    personal: "About you",
    fullName: "Full name",
    birthDate: "Date of birth",
    city: "City",
    profession: "Profession",
    years: "Years of experience",
    work: "Where you have worked",
    organization: "Gym, club or company",
    role: "Role",
    from: "From (year)",
    to: "To (year)",
    toHint: "Leave “To” empty if you still work there.",
    addWork: "Add a workplace",
    education: "Education and courses",
    institution: "School or institution",
    qualification: "Degree, course or certificate",
    year: "Year",
    addEducation: "Add a qualification",
    remove: "Remove",
    about: "About you as a trainer",
    aboutHint: "Your approach, specialities and the people you train (50 to 4000 characters).",
    documents: "Documents",
    documentsLead: "PDF, JPEG or PNG, up to 10 MB each and 6 in total. Certification and identification are required.",
    kinds: {
      certification: "Certifications and diplomas (required)",
      identification: "Identification document (required)",
      insurance: "Professional insurance (optional)",
      other: "Other evidence (optional)"
    } as Record<Kind, string>,
    choose: "Choose files",
    integrity:
      "Documents are checked for signs of being created or altered with artificial intelligence, for unsafe content and for reuse by other accounts. Refused files must be replaced with the original documents. False documents lead to the refusal of the application.",
    consent: "I confirm the information and the documents are genuine and mine.",
    submit: "Submit application",
    submitting: "Submitting…",
    fixFields: "Check the highlighted fields.",
    needDocuments: "Attach at least one certification and your identification document.",
    tooMany: "You can attach up to 6 documents.",
    tooLarge: "Each document can have up to 10 MB.",
    refused: "Some documents were refused. Replace them with the original documents:",
    reasons: {
      ai_provenance: "the file declares it was generated with AI",
      ai_generator_metadata: "the file carries the marks of an AI generator",
      unsafe_pdf_content: "the PDF contains scripts or embedded files",
      duplicate_document: "this file was already submitted by another account"
    } as Record<string, string>,
    genericError: "We could not submit your application. Please try again.",
    typeError: "Documents must be PDF, JPEG or PNG files."
  },
  pt: {
    title: "Candidatura a treinador RYZE",
    lead: "Diz-nos quem és e mostra as tuas qualificações. Cada candidatura é verificada automaticamente e revista por uma pessoa antes de qualquer decisão.",
    personal: "Sobre ti",
    fullName: "Nome completo",
    birthDate: "Data de nascimento",
    city: "Cidade",
    profession: "Profissão",
    years: "Anos de experiência",
    work: "Onde trabalhaste",
    organization: "Ginásio, clube ou empresa",
    role: "Função",
    from: "Desde (ano)",
    to: "Até (ano)",
    toHint: "Deixa “Até” vazio se ainda trabalhas lá.",
    addWork: "Adicionar um local de trabalho",
    education: "Formação e cursos",
    institution: "Escola ou instituição",
    qualification: "Curso, licenciatura ou certificado",
    year: "Ano",
    addEducation: "Adicionar uma formação",
    remove: "Remover",
    about: "Sobre ti como treinador",
    aboutHint: "A tua abordagem, especialidades e as pessoas que treinas (50 a 4000 caracteres).",
    documents: "Documentos",
    documentsLead: "PDF, JPEG ou PNG, até 10 MB cada e 6 no total. Certificação e identificação são obrigatórias.",
    kinds: {
      certification: "Certificados e diplomas (obrigatório)",
      identification: "Documento de identificação (obrigatório)",
      insurance: "Seguro profissional (opcional)",
      other: "Outras provas (opcional)"
    } as Record<Kind, string>,
    choose: "Escolher ficheiros",
    integrity:
      "Os documentos são verificados quanto a sinais de terem sido criados ou alterados com inteligência artificial, a conteúdo perigoso e à reutilização por outras contas. Os ficheiros recusados têm de ser substituídos pelos documentos originais. Documentos falsos levam à recusa da candidatura.",
    consent: "Confirmo que a informação e os documentos são verdadeiros e meus.",
    submit: "Enviar candidatura",
    submitting: "A enviar…",
    fixFields: "Confirma os campos assinalados.",
    needDocuments: "Anexa pelo menos um certificado e o teu documento de identificação.",
    tooMany: "Podes anexar até 6 documentos.",
    tooLarge: "Cada documento pode ter até 10 MB.",
    refused: "Alguns documentos foram recusados. Substitui-os pelos documentos originais:",
    reasons: {
      ai_provenance: "o ficheiro declara ter sido gerado com IA",
      ai_generator_metadata: "o ficheiro tem as marcas de um gerador de IA",
      unsafe_pdf_content: "o PDF contém scripts ou ficheiros embutidos",
      duplicate_document: "este ficheiro já foi submetido por outra conta"
    } as Record<string, string>,
    genericError: "Não foi possível enviar a candidatura. Tenta de novo.",
    typeError: "Os documentos têm de ser ficheiros PDF, JPEG ou PNG."
  }
} as const;

interface WorkRow {
  organization: string;
  role: string;
  start: string;
  end: string;
}

interface EducationRow {
  institution: string;
  qualification: string;
  year: string;
}

// TrainerApplicationForm is the complete trainer application (owner request,
// 2026-11-20), laid out like the contact page's form: the applicant's details,
// their background, a free text and the evidence files, sent together. The
// server validates every field and screens every file; nothing here decides.
export const TrainerApplicationForm = ({ defaultName, onSubmitted }: { defaultName: string; onSubmitted: () => void }) => {
  const { locale } = useI18n();
  const text = COPY[locale];
  const ids = useId();
  const [fullName, setFullName] = useState(defaultName);
  const [birthDate, setBirthDate] = useState("");
  const [city, setCity] = useState("");
  const [profession, setProfession] = useState("");
  const [years, setYears] = useState("");
  const [work, setWork] = useState<WorkRow[]>([{ organization: "", role: "", start: "", end: "" }]);
  const [education, setEducation] = useState<EducationRow[]>([{ institution: "", qualification: "", year: "" }]);
  const [about, setAbout] = useState("");
  const [files, setFiles] = useState<Record<Kind, File[]>>({ certification: [], identification: [], insurance: [], other: [] });
  const [consent, setConsent] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [invalid, setInvalid] = useState<string[]>([]);
  const [message, setMessage] = useState("");
  const [rejections, setRejections] = useState<TrainerDocumentRejection[]>([]);

  const totalFiles = Object.values(files).reduce((sum, list) => sum + list.length, 0);
  const bad = (field: string) => invalid.includes(field);

  const addFiles = (kind: Kind) => (event: ChangeEvent<HTMLInputElement>) => {
    const chosen = Array.from(event.target.files ?? []);
    event.target.value = "";
    if (chosen.some((file) => file.size > MAX_FILE_BYTES)) {
      setMessage(text.tooLarge);
      return;
    }
    const next = kind === "identification" ? chosen.slice(0, 1) : [...files[kind], ...chosen];
    const others = totalFiles - files[kind].length;
    if (others + next.length > MAX_FILES) {
      setMessage(text.tooMany);
      return;
    }
    setMessage("");
    setRejections([]);
    setFiles({ ...files, [kind]: next });
  };

  const removeFile = (kind: Kind, index: number) => setFiles({ ...files, [kind]: files[kind].filter((_, position) => position !== index) });

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (files.certification.length === 0 || files.identification.length === 0) {
      setMessage(text.needDocuments);
      return;
    }
    setSubmitting(true);
    setMessage("");
    setInvalid([]);
    setRejections([]);
    const workHistory: TrainerWorkEntry[] = work.map((row) => ({
      organization: row.organization,
      role: row.role,
      start_year: Number(row.start) || 0,
      end_year: row.end.trim() ? Number(row.end) || 0 : null
    }));
    const educationList: TrainerEducationEntry[] = education.map((row) => ({
      institution: row.institution,
      qualification: row.qualification,
      year: Number(row.year) || 0
    }));
    const upload: TrainerApplicationFiles = files;
    try {
      await submitTrainerApplication(
        {
          full_name: fullName,
          birth_date: birthDate,
          city,
          profession,
          years_experience: years.trim() === "" ? -1 : Number(years),
          work_history: workHistory,
          education: educationList,
          about
        },
        upload
      );
      onSubmitted();
    } catch (error) {
      if (error instanceof ApiError) {
        if (error.code === "VALIDATION_ERROR" && error.details.length > 0) {
          setInvalid(error.details);
          setMessage(text.fixFields);
        } else if (error.code === "DOCUMENTS_REJECTED") {
          const list = (error.payload.rejections as TrainerDocumentRejection[] | undefined) ?? [];
          setRejections(list);
          setMessage(text.refused);
        } else if (error.code === "DOCUMENTS_REQUIRED") {
          setMessage(text.needDocuments);
        } else if (error.code === "DOCUMENT_TOO_LARGE") {
          setMessage(text.tooLarge);
        } else if (error.code === "DOCUMENT_TYPE_NOT_ALLOWED") {
          setMessage(text.typeError);
        } else if (error.code === "ALREADY_TRAINER" || error.code === "APPLICATION_ALREADY_EXISTS") {
          onSubmitted();
        } else {
          setMessage(text.genericError);
        }
      } else {
        setMessage(text.genericError);
      }
    } finally {
      setSubmitting(false);
    }
  };

  const field = (id: string, label: string, error: boolean, input: ReactElement) => (
    <label className={joinClassNames(styles.field, error && styles.fieldError)} htmlFor={`${ids}-${id}`}>
      <span>{label}</span>
      {input}
    </label>
  );

  return (
    <form className={styles.form} onSubmit={(event) => void submit(event)} noValidate>
      <header className={styles.head}>
        <h1 className={styles.title}>{text.title}</h1>
        <p className={styles.lead}>{text.lead}</p>
      </header>

      <fieldset className={styles.group}>
        <legend>{text.personal}</legend>
        <div className={styles.grid}>
          {field("name", text.fullName, bad("full_name"), (
            <input id={`${ids}-name`} value={fullName} maxLength={120} autoComplete="name" required aria-invalid={bad("full_name")} onChange={(event) => setFullName(event.target.value)} />
          ))}
          {field("birth", text.birthDate, bad("birth_date"), (
            <input id={`${ids}-birth`} type="date" value={birthDate} autoComplete="bday" required aria-invalid={bad("birth_date")} onChange={(event) => setBirthDate(event.target.value)} />
          ))}
          {field("city", text.city, bad("city"), (
            <input id={`${ids}-city`} value={city} maxLength={80} autoComplete="address-level2" required aria-invalid={bad("city")} onChange={(event) => setCity(event.target.value)} />
          ))}
          {field("profession", text.profession, bad("profession"), (
            <input id={`${ids}-profession`} value={profession} maxLength={120} autoComplete="organization-title" required aria-invalid={bad("profession")} onChange={(event) => setProfession(event.target.value)} />
          ))}
          {field("years", text.years, bad("years_experience"), (
            <input id={`${ids}-years`} type="number" inputMode="numeric" min={0} max={60} value={years} required aria-invalid={bad("years_experience")} onChange={(event) => setYears(event.target.value)} />
          ))}
        </div>
      </fieldset>

      <fieldset className={joinClassNames(styles.group, bad("work_history") && styles.groupError)}>
        <legend>{text.work}</legend>
        {work.map((row, index) => (
          <div key={index} className={styles.entry}>
            <label className={styles.entryField}>
              <span>{text.organization}</span>
              <input value={row.organization} maxLength={120} onChange={(event) => setWork(work.map((item, position) => (position === index ? { ...item, organization: event.target.value } : item)))} />
            </label>
            <label className={styles.entryField}>
              <span>{text.role}</span>
              <input value={row.role} maxLength={120} onChange={(event) => setWork(work.map((item, position) => (position === index ? { ...item, role: event.target.value } : item)))} />
            </label>
            <label className={styles.entryField}>
              <span>{text.from}</span>
              <input type="number" inputMode="numeric" min={1950} max={THIS_YEAR} value={row.start} onChange={(event) => setWork(work.map((item, position) => (position === index ? { ...item, start: event.target.value } : item)))} />
            </label>
            <label className={styles.entryField}>
              <span>{text.to}</span>
              <input type="number" inputMode="numeric" min={1950} max={THIS_YEAR} value={row.end} onChange={(event) => setWork(work.map((item, position) => (position === index ? { ...item, end: event.target.value } : item)))} />
            </label>
            {work.length > 1 ? (
              <button type="button" className={styles.iconButton} aria-label={text.remove} onClick={() => setWork(work.filter((_, position) => position !== index))}>
                <Trash2 aria-hidden="true" />
              </button>
            ) : null}
          </div>
        ))}
        <p className={styles.hint}>{text.toHint}</p>
        {work.length < 10 ? (
          <Button type="button" variant="ghost" size="small" icon={<Plus aria-hidden="true" />} iconPosition="left" onClick={() => setWork([...work, { organization: "", role: "", start: "", end: "" }])}>
            {text.addWork}
          </Button>
        ) : null}
      </fieldset>

      <fieldset className={joinClassNames(styles.group, bad("education") && styles.groupError)}>
        <legend>{text.education}</legend>
        {education.map((row, index) => (
          <div key={index} className={joinClassNames(styles.entry, styles.entryThree)}>
            <label className={styles.entryField}>
              <span>{text.institution}</span>
              <input value={row.institution} maxLength={120} onChange={(event) => setEducation(education.map((item, position) => (position === index ? { ...item, institution: event.target.value } : item)))} />
            </label>
            <label className={styles.entryField}>
              <span>{text.qualification}</span>
              <input value={row.qualification} maxLength={120} onChange={(event) => setEducation(education.map((item, position) => (position === index ? { ...item, qualification: event.target.value } : item)))} />
            </label>
            <label className={styles.entryField}>
              <span>{text.year}</span>
              <input type="number" inputMode="numeric" min={1950} max={THIS_YEAR + 1} value={row.year} onChange={(event) => setEducation(education.map((item, position) => (position === index ? { ...item, year: event.target.value } : item)))} />
            </label>
            {education.length > 1 ? (
              <button type="button" className={styles.iconButton} aria-label={text.remove} onClick={() => setEducation(education.filter((_, position) => position !== index))}>
                <Trash2 aria-hidden="true" />
              </button>
            ) : null}
          </div>
        ))}
        {education.length < 10 ? (
          <Button type="button" variant="ghost" size="small" icon={<Plus aria-hidden="true" />} iconPosition="left" onClick={() => setEducation([...education, { institution: "", qualification: "", year: "" }])}>
            {text.addEducation}
          </Button>
        ) : null}
      </fieldset>

      <fieldset className={joinClassNames(styles.group, bad("about") && styles.groupError)}>
        <legend>{text.about}</legend>
        <textarea
          id={`${ids}-about`}
          className={styles.textarea}
          value={about}
          maxLength={4000}
          rows={6}
          aria-describedby={`${ids}-about-hint`}
          aria-invalid={bad("about")}
          onChange={(event) => setAbout(event.target.value)}
        />
        <p id={`${ids}-about-hint`} className={styles.hint}>
          {text.aboutHint} · {about.trim().length}/4000
        </p>
      </fieldset>

      <fieldset className={styles.group}>
        <legend>{text.documents}</legend>
        <p className={styles.hint}>{text.documentsLead}</p>
        {(Object.keys(text.kinds) as Kind[]).map((kind) => (
          <div key={kind} className={styles.upload}>
            <span className={styles.uploadLabel}>{text.kinds[kind]}</span>
            <label className={styles.uploadButton}>
              <FileUp aria-hidden="true" /> {text.choose}
              <input type="file" accept={ACCEPT} multiple={kind !== "identification"} onChange={addFiles(kind)} />
            </label>
            {files[kind].length > 0 ? (
              <ul className={styles.files}>
                {files[kind].map((file, index) => (
                  <li key={`${file.name}-${index}`}>
                    <span>{file.name}</span>
                    <button type="button" className={styles.iconButton} aria-label={`${text.remove} ${file.name}`} onClick={() => removeFile(kind, index)}>
                      <X aria-hidden="true" />
                    </button>
                  </li>
                ))}
              </ul>
            ) : null}
          </div>
        ))}
        <p className={styles.integrity}>
          <ShieldCheck aria-hidden="true" /> {text.integrity}
        </p>
      </fieldset>

      <label className={styles.consent}>
        <input type="checkbox" checked={consent} onChange={(event) => setConsent(event.target.checked)} />
        <span>{text.consent}</span>
      </label>

      <div aria-live="polite">
        {message ? (
          <div className={styles.alert} role="alert">
            <p>
              <AlertTriangle aria-hidden="true" /> {message}
            </p>
            {rejections.length > 0 ? (
              <ul>
                {rejections.map((rejection) => (
                  <li key={rejection.filename}>
                    <strong>{rejection.filename}</strong>: {rejection.reasons.map((reason) => text.reasons[reason] ?? reason).join("; ")}
                  </li>
                ))}
              </ul>
            ) : null}
          </div>
        ) : null}
      </div>

      <div>
        <Button type="submit" variant="primary" icon={<Send aria-hidden="true" />} disabled={submitting || !consent}>
          {submitting ? text.submitting : text.submit}
        </Button>
      </div>
    </form>
  );
};
