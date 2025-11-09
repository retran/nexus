import React, { useState, useEffect } from 'react';
import { Camera } from 'lucide-react';
import {
  glassPanelMaterial,
  glassTextSecondary,
  glassLink,
} from '@/lib/glass-styles';
import { cn } from '@/lib/utils';

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
        className="filter-[blur(4px)] dark:filter-[blur(4px)] fixed inset-0 z-0 bg-cover bg-center bg-no-repeat"
        style={{
          backgroundImage: currentPhoto
            ? `url(${currentPhoto.imageUrl})`
            : undefined,
          transform: 'scale(1.1)',
        }}
        aria-hidden="true"
      />
      {currentPhoto && (
        <div
          className={cn(
            'fixed bottom-3 left-3 z-10 flex items-center gap-2 rounded-2xl px-3 py-2 shadow-xl',
            glassPanelMaterial
          )}
        >
          <Camera className="h-3 w-3 shrink-0 opacity-60" />
          <span
            className={cn(
              'text-xs leading-tight opacity-80',
              glassTextSecondary
            )}
          >
            {currentPhoto.source === 'unsplash' ? (
              <>
                Photo by{' '}
                <a
                  href={currentPhoto.photographerLink}
                  target="_blank"
                  rel="noopener noreferrer"
                  className={cn(glassLink, 'font-semibold')}
                >
                  {currentPhoto.photographerName}
                </a>
                {' / '}
                <a
                  href="https://unsplash.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  className={cn(glassLink, 'opacity-90')}
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
                  className={cn(glassLink, 'opacity-90')}
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
