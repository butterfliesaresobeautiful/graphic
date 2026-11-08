"""Plot actual Go results for the report; no image-processing implementation here."""
import os
os.environ.setdefault('MPLCONFIGDIR','/tmp/lb8-matplotlib')
from pathlib import Path
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import numpy as np
from PIL import Image

ROOT=Path(__file__).resolve().parents[1]
OUT=ROOT/'report'
plt.rcParams.update({'font.family':'DejaVu Sans','font.size':11,'axes.spines.top':False,'axes.spines.right':False})
methods=['nearest','bilinear','bicubic']
labels=['Ближайший сосед','Билинейная','Бикубическая']
def read(p,m):return np.array(Image.open(ROOT/'images'/p/(m+'.png')).convert('RGB'))
def comparison(name,presets,titles,zoom=False):
 fig,axs=plt.subplots(len(presets),3,figsize=(12,3.7*len(presets)),squeeze=False,layout='constrained')
 for row,(p,title) in enumerate(zip(presets,titles)):
  for col,(m,label) in enumerate(zip(methods,labels)):
   data=read(p,m);ax=axs[row,col]
   if zoom:
    h,w=data.shape[:2];x,y=w//2-48,h//2-36;data=data[y:y+72,x:x+96]
    subtitle=f'{title}\nROI [{x},{y}; {x+96},{y+72})'
   else:subtitle=title
   ax.imshow(data,interpolation='nearest');ax.axis('off');ax.set_title(label+'\n'+subtitle,fontsize=10)
 fig.savefig(OUT/(name+'.png'),dpi=180,facecolor='white');plt.close(fig)
comparison('comparison-scale',['scale-up','scale-down'],['Sx=1.7; Sy=1.4','Sx=0.55; Sy=0.65'])
comparison('comparison-rotation',['rotate-positive','rotate-negative'],['θ=+28°','θ=−17°'])
comparison('comparison-shear',['shear-positive','shear-negative'],['hx=0.45; hy=0.15','hx=−0.35; hy=0.10'])
comparison('comparison-combined',['combined'],['Sx=1.25; Sy=0.9; θ=28°; hx=0.25; hy=−0.1'])
comparison('comparison-zoom',['scale-up','combined'],['Увеличение','Составное преобразование'],zoom=True)
fig,ax=plt.subplots(figsize=(10,3.5),layout='constrained')
for m,l in zip(methods,labels):
 data=read('scale-up',m);ax.plot(np.arange(30,86),data[84,30:86,0],label=l,linewidth=1.5)
ax.set(xlabel='Столбец выходного изображения, u',ylabel='Красный канал, 0…255',title='Измеренный профиль: увеличение 1.7 × 1.4, строка v=84',ylim=(-5,260));ax.grid(alpha=.25);ax.legend(ncol=3,fontsize=10)
fig.savefig(OUT/'profile.png',dpi=180,facecolor='white');plt.close(fig)
t=np.linspace(0,1,201);ws=[-t*(t-1)*(t-2)/6,(t+1)*(t-1)*(t-2)/2,-(t+1)*t*(t-2)/2,(t+1)*t*(t-1)/6]
fig,ax=plt.subplots(figsize=(9,3.5),layout='constrained')
for w,l in zip(ws,['w(−1,t)','w(0,t)','w(1,t)','w(2,t)']):ax.plot(t,w,label=l)
ax.axhline(0,color='#777777',linewidth=.7);ax.grid(alpha=.25);ax.set(xlabel='Дробная координата t',ylabel='Вес',title='Кубические веса Лагранжа для четырёх соседних узлов');ax.legend(ncol=4)
fig.savefig(OUT/'weights.png',dpi=180,facecolor='white');plt.close(fig)
print('Created seven figures from saved Go PNGs and the analytical weights.')
