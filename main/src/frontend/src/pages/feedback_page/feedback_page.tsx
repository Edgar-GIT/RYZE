import { ArrowRight } from "lucide-react";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { Reveal } from "@/components/reveal/reveal";
import { BRAND_ASSETS } from "@/constants/brand_assets";
import feedbackBackground from "@resources/img/feedbacks/feedback_bg.png";
import feedbackOne from "@resources/img/feedbacks/f1.jpeg";
import feedbackTwo from "@resources/img/feedbacks/f2.jpeg";
import feedbackThree from "@resources/img/feedbacks/f3.jpeg";

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

const Stars = () => (
  <span className={styles.stars} role="img" aria-label="Rated 5 out of 5 stars">
    ★★★★★
  </span>
);

export const FeedbackPage = () => (
  <PageWrapper>
    <section
      className={styles.hero}
      style={{ backgroundImage: `url(${feedbackBackground})` }}
      aria-label="Feedback"
    >
      <Container className={styles.inner}>
        <div className={styles.topRow}>
          <div className={styles.copy}>
            <Reveal y={18}>
              <p className={styles.eyebrow}>Feedback</p>
            </Reveal>

            <Reveal y={18} delay={0.08}>
              <h1 className={styles.title}>
                Real people.
                <br />
                <span className={styles.accent}>Real results.</span>
              </h1>
            </Reveal>

            <Reveal y={18} delay={0.16}>
              <p className={styles.description}>
                Thousands of people are already transforming their lives with
                RYZE. Here are some of their stories.
              </p>
            </Reveal>
          </div>

          <Reveal className={styles.visual} y={18} delay={0.12}>
            <img src={BRAND_ASSETS.slogan} alt="RYZE" className={styles.visualLogo} />
          </Reveal>
        </div>

        <div className={styles.grid}>
          {testimonials.map((testimonial, index) => (
            <Reveal
              className={styles.gridItem}
              y={26}
              delay={0.15 + index * 0.12}
              key={testimonial.alt}
            >
              <article className={styles.card}>
                <div className={styles.media}>
                  <img src={testimonial.image} alt={testimonial.alt} loading="lazy" />
                </div>
                <div className={styles.body}>
                  <div className={styles.starsRow}>
                    <span className={styles.quoteMark} aria-hidden="true">
                      “
                    </span>
                    <Stars />
                  </div>
                  <p className={styles.name}>Anonymous</p>
                  <p className={styles.quote}>{testimonial.quote}</p>
                </div>
              </article>
            </Reveal>
          ))}
        </div>

        <Reveal className={styles.ctaReveal} y={18} delay={0.22}>
          <div className={styles.cta}>
            <Button
              to="/register"
              size="large"
              variant="secondary"
              className={styles.ctaButton}
              icon={<ArrowRight />}
            >
              START YOUR TRANSFORMATION
            </Button>
          </div>
        </Reveal>
      </Container>
    </section>
  </PageWrapper>
);
