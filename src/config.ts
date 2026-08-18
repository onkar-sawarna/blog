// Everything personal to the site lives here. Edit this file, not the layouts.

export const SITE = {
  title: 'Onkar Sawarna',
  tagline: 'engineering, networking, observability, and distributed systems',
  author: 'Onkar Sawarna',
  description:
    'Software engineer at Acceldata. Notes on observability, networking, and distributed systems.',
  // Set this to your real domain before deploying — RSS and sitemap use it.
  url: 'https://www.onkarsawarna.dev',
  resume: '/resume.pdf',
  // Buttondown username. Form posts to their public embed endpoint.
  newsletter: 'onkarsawarna',
};

// Delete any you don't want; the footer renders whatever is left.
// Paid PDFs. buyUrl must be a Payment Link created with callback_url
// https://www.onkarsawarna.dev/notes/thanks and callback_method get
// (go run ./cmd/razorpay-link). Dashboard links that do not redirect
// cannot auto-deliver. Leave buyUrl empty to fall back to mailto.
export const NOTES: Array<{
  id: string;
  title: string;
  description: string;
  price: string;
  pages?: number;
  preview?: string;
  buyUrl?: string;
}> = [
  {
    id: 'computer-networks',
    title: 'Computer networks, as they show up on a box',
    description:
      'Computer networks from first words: process, packet, IP, port, then tuples, NAT, ping, and what to measure when a connect fails. My models, not a syllabus.',
    price: '₹1',
    pages: 20,
    preview: '/notes/computer-networks-preview.png',
    buyUrl: 'https://rzp.io/rzp/SKal7Wl',
  },
];

export const SOCIALS = [
  { label: 'GitHub', href: 'https://github.com/onkar-sawarna' },
  { label: 'LinkedIn', href: 'https://www.linkedin.com/in/onkar-sawarna-569615187/' },
  { label: 'X', href: 'https://x.com/onkar_sawarna' },
  { label: 'Email', href: 'mailto:onkarsawarna@gmail.com' },
];
