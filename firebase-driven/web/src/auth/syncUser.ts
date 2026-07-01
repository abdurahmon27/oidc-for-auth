import { doc, serverTimestamp, setDoc } from 'firebase/firestore';
import type { User as FirebaseUser } from 'firebase/auth';
import { db } from '../firebase';
import { normalizeProviderId } from '../types/auth';

export async function syncUser(fbUser: FirebaseUser): Promise<void> {
  try {
    // Custom-token (Telegram) sessions have no providerData; the
    // telegramVerify function already recorded providers: ['telegram'], so
    // don't clobber it with an empty array here.
    const providerIds = fbUser.providerData.map((p) => normalizeProviderId(p.providerId));
    const merge: Record<string, unknown> = {
      email: fbUser.email ?? null,
      name: fbUser.displayName ?? null,
      avatar_url: fbUser.photoURL ?? null,
      phone: fbUser.phoneNumber ?? null,
      updatedAt: serverTimestamp(),
    };
    if (providerIds.length > 0) merge.providers = providerIds;
    await setDoc(
      doc(db, 'users', fbUser.uid),
      merge,
      { merge: true }
    );
  } catch (err) {
    console.error('syncUser failed:', err);
  }
}
