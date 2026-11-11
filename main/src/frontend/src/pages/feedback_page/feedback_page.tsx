import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { ArrowRight } from "lucide-react";

import { BrandStop, PublicAction, Waves } from "@/components/public_ui/public_ui";
import ui from "@/components/public_ui/public_ui.module.css";
import { useMediaQuery } from "@/hooks/use_media_query";
import { joinClassNames } from "@utils/class_names";
import feedbackOne from "@resources/img/feedbacks/f1.jpeg";
import feedbackTwo from "@resources/img/feedbacks/f2.jpeg";
import feedbackThree from "@resources/img/feedbacks/f3.jpeg";

import { CommunityFeedback } from "./community_feedback";
import styles from "./feedback_page.module.css";

interface Testimonial {
  image: string;
  alt: string;
  quote: string;
}

const testimonials: Testimonial[] = [
  {
    image: feedbackOne,
    alt: "RYZE member who gained 12kg and rebuilt their confidence in 16 weeks",
    quote:
      "Over 16 weeks, I gained 12kg and a whole new level of confidence. The RYZE training and nutrition plan completely changed the way I approach fitness."
  },
  {
    image: feedbackTwo,
    alt: "RYZE member who lost 9kg and feels more energetic",
    quote:
      "I lost 9kg and feel so much more energetic! I learned how to train with purpose and eat better. I feel stronger both inside and out."
  },
  {
    image: feedbackThree,
    alt: "RYZE member who rebuilt their confidence sustainably",
    quote:
      "RYZE helped me rebuild my confidence and feel better in my own body. The results came consistently and in a healthy, sustainable way."
  }
];

// Below 768px the stories are a horizontal scroller (see the stylesheet), which
// must be keyboard-focusable; from 768px they are a static grid.
const STORIES_SCROLL_QUERY = "(max-width: 767px)";

export const FeedbackPage = () => {
  const storiesScroll = useMediaQuery(STORIES_SCROLL_QUERY);

  return (
    <PageWrapper className={ui.page}>
      <section className={joinClassNames(ui.toneNavy, styles.header)} aria-labelledby="feedback-heading">
        <div className={joinClassNames(ui.container, styles.headerLayout)}>
          <div className={styles.headerCopy}>
            <p className={ui.label}>Feedback</p>
            <h1 id="feedback-heading" className={ui.display}>
              <span className={styles.titleLine}>Real people.</span>{" "}
              <span className={joinClassNames(styles.titleLine, ui.hl)}>Real results.</span>
            </h1>
            <p className={ui.lead}>
              Thousands of people are already transforming their lives with RYZE. Here are some of
              their stories.
            </p>
          </div>
        </div>
      </section>

      <section className={joinClassNames(ui.toneNavy, styles.stories)} aria-label="Member stories">
        <div className={ui.container}>
          <ul
            className={styles.track}
            tabIndex={storiesScroll ? 0 : undefined}
            aria-label="Member stories"
          >
            {testimonials.map((testimonial) => (
              <li className={joinClassNames(ui.surface, styles.card)} key={testimonial.alt}>
                <img
                  className={styles.media}
                  src={testimonial.image}
                  alt={testimonial.alt}
                  width="1080"
                  height="1350"
                  loading="lazy"
                  decoding="async"
                />
                <figure className={styles.body}>
                  <p className={styles.stars} role="img" aria-label="Rated 5 out of 5 stars">
                    ★★★★★
                  </p>
                  <blockquote className={styles.quote}>
                    <p>{testimonial.quote}</p>
                  </blockquote>
                  <figcaption className={styles.author}>Anonymous</figcaption>
                </figure>
              </li>
            ))}
          </ul>
          <p className={joinClassNames(ui.caption, styles.swipeHint)}>Swipe to read more stories.</p>
        </div>
      </section>

      <CommunityFeedback />

      <section className={joinClassNames(ui.toneNavy, styles.cta)} aria-labelledby="feedback-cta-heading">
        <div className={joinClassNames(ui.container, ui.toneLight, ui.lightPanel, styles.ctaLayout)}>
          <Waves />
          <h2 id="feedback-cta-heading" className={ui.heading}>
            Your story starts with <span className={ui.hl}>one session</span>
            <BrandStop />
          </h2>
          <PublicAction to="/register" variant="ink" icon={<ArrowRight />} className={styles.ctaAction}>
            Start your transformation
          </PublicAction>
        </div>
      </section>
    </PageWrapper>
  );
};
