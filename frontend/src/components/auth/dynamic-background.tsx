import React, { useState, useEffect } from 'react';
import { Camera } from 'lucide-react';

interface PhotoData {
  imageUrl: string;
  photographerName: string;
  photographerLink: string;
  source: string;
}

interface PhotosResponse {
  light: PhotoData;
  dark: PhotoData;
}

export const DynamicBackground: React.FC = () => {
  const [photos, setPhotos] = useState<PhotosResponse | null>(null);
  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
    const fetchPhotos = async () => {
      try {
        const response = await fetch('http://api.nexus.local/photos');
        if (response.ok) {
          const data: PhotosResponse = await response.json();
          setPhotos(data);
        }
      } catch (error) {
        console.error('Failed to fetch photos:', error);
      }
    };

    fetchPhotos();

    // Check for theme changes
    const checkTheme = () => {
      setIsDark(document.documentElement.classList.contains('dark'));
    };

    checkTheme();

    // Listen for theme changes
    const observer = new MutationObserver(checkTheme);
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    });

    return () => observer.disconnect();
  }, []);

  const currentPhoto = photos ? (isDark ? photos.dark : photos.light) : null;

  return (
    <>
      <div
        className="filter-[blur(4px)_brightness(0.8)] dark:filter-[blur(4px)_brightness(0.4)] fixed inset-0 z-0 bg-cover bg-center bg-no-repeat"
        style={{
          backgroundImage: currentPhoto
            ? `url(${currentPhoto.imageUrl})`
            : undefined,
          transform: 'scale(1.1)',
        }}
        aria-hidden="true"
      />
      {currentPhoto && (
        <div className="fixed bottom-3 left-3 z-10 flex items-center gap-2 rounded-2xl border border-white/10 bg-white/40 px-3 py-2 shadow-xl backdrop-blur-xl dark:bg-black/40">
          <Camera className="h-3 w-3 shrink-0 opacity-60" />
          <span className="text-xs leading-tight text-black/80 opacity-80 dark:text-white/80">
            {currentPhoto.source === 'unsplash' ? (
              <>
                Photo by{' '}
                <a
                  href={currentPhoto.photographerLink}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-semibold underline decoration-1 underline-offset-2 transition-colors hover:text-black dark:hover:text-white"
                >
                  {currentPhoto.photographerName}
                </a>
                {' / '}
                <a
                  href="https://unsplash.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="underline decoration-1 underline-offset-2 opacity-90 transition-colors hover:text-black dark:hover:text-white"
                >
                  Unsplash
                </a>
              </>
            ) : (
              <>
                Photo /{' '}
                <a
                  href="https://picsum.photos"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="underline decoration-1 underline-offset-2 opacity-90 transition-colors hover:text-black dark:hover:text-white"
                >
                  Picsum
                </a>
              </>
            )}
          </span>
        </div>
      )}
    </>
  );
};
